package nativersc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/polagonow/pola/core"
	reactrenderer "github.com/polagonow/pola/renderer/react"
	"github.com/polagonow/pola/serveraction"
	"github.com/polagonow/pola/shell"

	gojalib "github.com/dop251/goja"
)

// Time helpers for testability.
var (
	timeNow   = time.Now
	timeSince = time.Since
)

// loopRenderer is implemented by JS runtimes (currently engine/goja) that allow a
// Go-driven renderer to run code on the event loop with direct runtime access.
// nativersc requires this because it walks the React element tree natively.
type loopRenderer interface {
	RunRender(func(rt *gojalib.Runtime) error) error
}

// Renderer is a Go-driven React Server Components renderer. It executes server
// components in goja but serializes the Flight wire format natively in Go.
type Renderer struct {
	pool core.SSRPool
	deps atomic.Pointer[core.RenderDeps]
}

// New creates a nativersc renderer.
func New() *Renderer { return &Renderer{} }

func (r *Renderer) Name() string             { return "nativersc" }
func (r *Renderer) FileExtensions() []string { return []string{".tsx", ".jsx", ".ts", ".js"} }
func (r *Renderer) Capabilities() []core.Capability {
	return []core.Capability{"streaming", "rsc"}
}

// SetRenderDeps implements core.RenderDepsAware.
func (r *Renderer) SetRenderDeps(deps core.RenderDeps) { r.deps.Store(&deps) }

func (r *Renderer) loadDeps() core.RenderDeps {
	if d := r.deps.Load(); d != nil {
		return *d
	}
	return core.RenderDeps{}
}

// LoadBundle implements core.BundleLoader. It asks the engine to create an
// SSRPool from the compiled server bundle (used only for VM lifecycle here).
func (r *Renderer) LoadBundle(engine core.JSEngine, bundle []byte) error {
	factory, ok := engine.(core.SSRPoolFactory)
	if !ok {
		return fmt.Errorf("nativersc renderer: engine %q does not implement SSRPoolFactory", engine.Name())
	}
	pool, err := factory.NewSSRPool(bundle)
	if err != nil {
		return fmt.Errorf("nativersc renderer: create SSR pool: %w", err)
	}
	r.pool = pool
	return nil
}

// ServeHTTP implements core.Renderer. It handles both RSC Flight requests
// (Content-Type: text/x-component) and HTML page loads.
func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	deps := r.loadDeps()
	ctx := req.Context()
	route, props, requestContext, status, injectors := core.RenderRequestFrom(ctx)

	isNotFound := route == nil
	if route == nil {
		if status == 0 {
			status = http.StatusNotFound
		}
		if deps.NotFoundRoute != nil {
			route = deps.NotFoundRoute
			if props == nil {
				props = map[string]any{"params": map[string]any{}, "searchParams": map[string]any{}}
			}
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			r.serveHTML(w, req, nil, &deps)
			return
		}
	}

	renderReq := core.RenderRequest{
		Route:          *route,
		Props:          props,
		RequestContext: requestContext,
		Injectors:      injectors,
	}

	if req.Header.Get("Content-Type") == reactrenderer.ContentType {
		r.serveFlight(ctx, w, req, renderReq, status, &deps)
		return
	}

	// HTML page load — check cache for inline SSR data.
	var ssrData []byte
	cacheKey := "ssr:" + req.URL.Path + "?" + req.URL.RawQuery
	if deps.Cache != nil {
		if cached, ok, err := deps.Cache.Get(ctx, cacheKey); err != nil {
			logError(deps.Logger, "pola: cache get", "key", cacheKey, "err", err)
		} else if ok {
			ssrData = cached
		}
	}
	// Pre-render only for not-found pages (the client won't request Flight for them).
	if ssrData == nil && isNotFound {
		if data, err := r.RenderToBytes(ctx, renderReq); err == nil {
			ssrData = data
			if deps.Cache != nil {
				if err := deps.Cache.Set(ctx, cacheKey, ssrData, core.CacheOptions{
					TTL: renderReq.Route.Revalidate,
				}); err != nil {
					logError(deps.Logger, "pola: cache set", "err", err)
				}
			}
		} else {
			logError(deps.Logger, "pola: render", "err", err)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	r.serveHTML(w, req, ssrData, &deps)
}

// serveFlight handles RSC Flight requests with caching and streaming.
func (r *Renderer) serveFlight(ctx context.Context, w http.ResponseWriter, req *http.Request, renderReq core.RenderRequest, status int, deps *core.RenderDeps) {
	w.Header().Set("Cache-Control", "no-store")

	if deps.Cache != nil {
		cacheKey := "ssr:" + req.URL.Path + "?" + req.URL.RawQuery
		if cached, ok, err := deps.Cache.Get(ctx, cacheKey); err != nil {
			logError(deps.Logger, "pola: cache get", "key", cacheKey, "err", err)
		} else if ok {
			w.Header().Set("Content-Type", reactrenderer.ContentType+"; charset=utf-8")
			w.WriteHeader(status)
			w.Write(cached) //nolint:errcheck
			return
		}
	}

	var tw *teeWriter
	if deps.Cache != nil {
		tw = &teeWriter{ResponseWriter: w}
		w = tw
	}

	if deps.Tracer != nil {
		var span core.Span
		ctx, span = deps.Tracer.StartSpan(ctx, "pola.render")
		defer span.End()
	}

	w.Header().Set("Content-Type", reactrenderer.ContentType+"; charset=utf-8")
	w.WriteHeader(status)

	renderStart := timeNow()
	err := r.RenderToWriter(ctx, renderReq, newStreamWriter(w))
	if deps.Metrics != nil {
		deps.Metrics.RecordRender(renderReq.Route.Pattern, timeSince(renderStart))
	}
	if err != nil {
		logError(deps.Logger, "pola: render", "err", err)
	}

	if tw != nil && len(tw.buf) > 0 {
		if err := deps.Cache.Set(ctx, "ssr:"+req.URL.Path+"?"+req.URL.RawQuery, tw.buf, core.CacheOptions{
			TTL: renderReq.Route.Revalidate,
		}); err != nil {
			logError(deps.Logger, "pola: cache set", "err", err)
		}
	}
}

// serveHTML returns the HTML shell for initial page loads.
func (r *Renderer) serveHTML(w http.ResponseWriter, req *http.Request, ssrData []byte, deps *core.RenderDeps) {
	params := core.ShellParams{
		Metadata:      defaultMetadata(),
		DocumentProps: deps.DocumentProps,
	}
	if deps.BundleOutput != nil {
		params.ImportURLs = deps.BundleOutput.ImportURLs
		params.ClientScript = deps.BundleOutput.ClientEntryURL
	}
	if len(deps.CSSURLs) > 0 {
		params.Stylesheets = deps.CSSURLs
	}
	if deps.DevScript != "" {
		params.Scripts = append(params.Scripts, deps.DevScript)
	}
	if nonce, ok := req.Context().Value(core.NonceContextKey).(string); ok && nonce != "" {
		params.Nonce = nonce
	}
	if token, ok := req.Context().Value(core.CSRFTokenContextKey).(string); ok && token != "" {
		if params.Metadata == nil {
			params.Metadata = &core.Metadata{}
		}
		if params.Metadata.Other == nil {
			params.Metadata.Other = make(map[string]string)
		}
		params.Metadata.Other["csrf-param"] = "authenticity_token"
		params.Metadata.Other["csrf-token"] = token
	}

	if len(ssrData) > 0 {
		encoded, err := json.Marshal(string(ssrData))
		if err != nil {
			logError(deps.Logger, "pola: marshal SSR data", "err", err)
		} else {
			params.Scripts = append(params.Scripts, fmt.Sprintf("self.%s=%s", reactrenderer.SSRData, encoded))
		}
	}

	var html string
	if deps.Shell != nil {
		html = deps.Shell.Render(params)
	} else {
		html = "<!DOCTYPE html><html><body></body></html>"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html)) //nolint:errcheck
}

// ── render core (Go-driven) ─────────────────────────────────────────────────

// RenderToWriter acquires a VM, performs a full Go-driven RSC Flight render, and
// streams all rows to w.
func (r *Renderer) RenderToWriter(ctx context.Context, req core.RenderRequest, w core.StreamWriter) error {
	fw := newFlightWriter(streamWriterIO{sw: w})
	if err := r.renderInto(ctx, req, fw); err != nil {
		return err
	}
	fw.flush()
	return nil
}

// RenderToBytes performs a full Go-driven RSC Flight render into a byte slice.
func (r *Renderer) RenderToBytes(ctx context.Context, req core.RenderRequest) ([]byte, error) {
	var buf bytes.Buffer
	fw := newFlightWriter(&buf)
	if err := r.renderInto(ctx, req, fw); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderInto runs the reconciler against a prepared VM, writing rows to fw.
func (r *Renderer) renderInto(ctx context.Context, req core.RenderRequest, fw *flightWriter) error {
	if r.pool == nil {
		return fmt.Errorf("nativersc renderer: VM pool not configured")
	}
	vm, propsJSON, err := r.prepareVM(ctx, req)
	if err != nil {
		return err
	}
	defer r.pool.Release(vm)

	lr, ok := vm.(loopRenderer)
	if !ok {
		return fmt.Errorf("nativersc renderer: engine runtime %T does not support RunRender (use the goja engine)", vm)
	}

	deps := r.loadDeps()
	var importURLs map[string]string
	if deps.BundleOutput != nil {
		importURLs = deps.BundleOutput.ImportURLs
	}

	// The reconciler orchestrates its own (sequential, non-nested) RunRender
	// calls so async server components can settle between runs.
	rec := newReconciler(lr, fw, importURLs, deps.Logger)
	return rec.render(req.Route.Export, propsJSON)
}

// prepareVM acquires a VM, applies injectors, sets the request context, and
// marshals props. The caller must release the VM when done.
func (r *Renderer) prepareVM(ctx context.Context, req core.RenderRequest) (core.SSRRuntime, string, error) {
	vm, err := r.pool.Acquire()
	if err != nil {
		return nil, "", fmt.Errorf("nativersc renderer: acquire VM: %w", err)
	}
	for _, inj := range req.Injectors {
		if err := inj.Inject(ctx, vm); err != nil {
			r.pool.Release(vm)
			return nil, "", fmt.Errorf("nativersc renderer: inject %s: %w", inj.Name(), err)
		}
	}
	if err := vm.SetRequestContext(req.RequestContext); err != nil {
		r.pool.Release(vm)
		return nil, "", fmt.Errorf("nativersc renderer: set context: %w", err)
	}
	propsJSON, err := json.Marshal(req.Props)
	if err != nil {
		r.pool.Release(vm)
		return nil, "", fmt.Errorf("nativersc renderer: marshal props: %w", err)
	}
	return vm, string(propsJSON), nil
}

// InvokeAction implements core.ServerActionInvoker. It acquires a VM, applies
// the per-request injectors and request context, then invokes the registered
// server action via the bundle's __invokeServerAction__ helper, awaiting its
// Promise. The helper returns a JSON envelope string which is decoded into the
// result. The VM is always released back to the pool.
func (r *Renderer) InvokeAction(ctx context.Context, in core.InvokeInput) (core.InvokeOutput, error) {
	return serveraction.Invoke(ctx, r.pool, in)
}

// ── pipeline hooks ──────────────────────────────────────────────────────────

// GenerateEntry implements the internal entryGenerator interface.
func (r *Renderer) GenerateEntry(discovery core.DiscoveryResult) (string, error) {
	gen := &EntryGenerator{}
	return gen.Generate(EntryGenConfig{
		Pages:                 discovery.Pages,
		AppDir:                discovery.AppDir,
		GlobalErrorPath:       discovery.GlobalError,
		GlobalNotFoundPath:    discovery.GlobalNotFound,
		RootLayoutReturnsHTML: discovery.RootLayoutReturnsHTML,
		ServerActions:         serveraction.Scan(discovery.AppDir),
	})
}

// ExtractDocumentProps calls __extractShell__ on a VM to get the root layout's
// HTML, returning nil when the root layout doesn't return <html>.
func (r *Renderer) ExtractDocumentProps() (*core.DocumentProps, error) {
	if r.pool == nil {
		return nil, nil
	}
	vm, err := r.pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("nativersc renderer: acquire VM for shell extraction: %w", err)
	}
	defer r.pool.Release(vm)

	result, err := vm.Call(reactrenderer.ExtractShellFn)
	if err != nil {
		return nil, nil
	}
	htmlStr, ok := result.(string)
	if !ok || htmlStr == "" {
		return nil, nil
	}
	return shell.ExtractDocumentProps(htmlStr)
}

// BundleConditions returns the esbuild conditions for the RSC server pass.
func (r *Renderer) BundleConditions() []string {
	return []string{"react-server", "browser", "module", "default"}
}

// ClientEntry returns the package specifier for the client hydration entry.
// nativersc reuses the React client (it parses the same Flight wire format).
func (r *Renderer) ClientEntry() string { return "@pola/react/client" }

// ── helpers (copied from renderer/react to keep that package untouched) ──────

func logError(logger core.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

func defaultMetadata() *core.Metadata {
	faviconURL := "/public/favicon.ico"
	return &core.Metadata{
		Title: core.Title{Default: "Pola"},
		Icons: &core.Icons{Icon: []core.Icon{{URL: faviconURL}}},
	}
}

// teeWriter captures written bytes while passing through to the underlying writer.
type teeWriter struct {
	http.ResponseWriter
	buf []byte
}

func (tw *teeWriter) Write(p []byte) (int, error) {
	tw.buf = append(tw.buf, p...)
	return tw.ResponseWriter.Write(p)
}

func (tw *teeWriter) Flush() {
	if f, ok := tw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// streamWriter adapts http.ResponseWriter to core.StreamWriter.
type streamWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newStreamWriter(w http.ResponseWriter) *streamWriter {
	sw := &streamWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		sw.flusher = f
	}
	return sw
}

func (sw *streamWriter) WriteRaw(p []byte) (int, error) { return sw.w.Write(p) }
func (sw *streamWriter) Flush() {
	if sw.flusher != nil {
		sw.flusher.Flush()
	}
}

// streamWriterIO adapts core.StreamWriter to io.Writer (+ Flush) for flightWriter.
type streamWriterIO struct{ sw core.StreamWriter }

func (a streamWriterIO) Write(p []byte) (int, error) { return a.sw.WriteRaw(p) }
func (a streamWriterIO) Flush()                       { a.sw.Flush() }

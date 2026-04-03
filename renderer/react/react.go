// Package react provides a React Server Components (RSC) renderer for Pola.
// It implements streaming SSR via the RSC Flight wire protocol.
package react

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
	"github.com/polagonow/pola/shell"
)

// Time helpers for testability.
var (
	timeNow   = time.Now
	timeSince = time.Since
)

// Renderer implements the React SSR renderer.
type Renderer struct {
	pool core.SSRPool
	deps atomic.Pointer[core.RenderDeps]
}

// New creates a React renderer.
func New() *Renderer { return &Renderer{} }

func (r *Renderer) Name() string             { return "react" }
func (r *Renderer) FileExtensions() []string { return []string{".tsx", ".jsx", ".ts", ".js"} }
func (r *Renderer) Capabilities() []core.Capability {
	return []core.Capability{"streaming", "rsc"}
}

// SetRenderDeps implements core.RenderDepsAware.
// It atomically swaps the deps pointer so that in-flight requests using the
// previous snapshot are unaffected.
func (r *Renderer) SetRenderDeps(deps core.RenderDeps) {
	r.deps.Store(&deps)
}

// loadDeps returns a consistent snapshot of render deps.
func (r *Renderer) loadDeps() core.RenderDeps {
	if d := r.deps.Load(); d != nil {
		return *d
	}
	return core.RenderDeps{}
}

// LoadBundle implements core.BundleLoader. It asks the engine to create an
// SSRPool from the compiled server bundle.
func (r *Renderer) LoadBundle(engine core.JSEngine, bundle []byte) error {
	factory, ok := engine.(core.SSRPoolFactory)
	if !ok {
		return fmt.Errorf("react renderer: engine %q does not implement SSRPoolFactory", engine.Name())
	}
	pool, err := factory.NewSSRPool(bundle)
	if err != nil {
		return fmt.Errorf("react renderer: create SSR pool: %w", err)
	}
	r.pool = pool
	return nil
}

// ContentType is the MIME type for the RSC Flight wire format.
const ContentType = "text/x-component"

// ServeHTTP implements core.Renderer. It handles both RSC Flight requests
// (Content-Type: text/x-component) and HTML page loads.
func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	deps := r.loadDeps()
	ctx := req.Context()
	route, props, requestContext, status, injectors := core.RenderRequestFrom(ctx)

	// When no route matched, substitute the not-found page if available.
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
			// No not-found page — serve a bare HTML shell.
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

	isFlight := req.Header.Get("Content-Type") == ContentType

	if isFlight {
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
	// If no cached data, try to pre-render.
	if ssrData == nil {
		if data, err := r.RenderToBytes(ctx, renderReq); err == nil {
			ssrData = data
			// Fill cache so subsequent requests are served instantly.
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
	// Cache integration for Flight requests.
	if deps.Cache != nil {
		cacheKey := "ssr:" + req.URL.Path + "?" + req.URL.RawQuery
		// Serve from cache if available — instant, no VM needed.
		if cached, ok, err := deps.Cache.Get(ctx, cacheKey); err != nil {
			logError(deps.Logger, "pola: cache get", "key", cacheKey, "err", err)
		} else if ok {
			w.Header().Set("Content-Type", ContentType+"; charset=utf-8")
			w.WriteHeader(status)
			w.Write(cached) //nolint:errcheck
			return
		}
	}

	// Tee output to fill cache for next request.
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

	w.Header().Set("Content-Type", ContentType+"; charset=utf-8")
	w.WriteHeader(status)

	renderStart := timeNow()
	err := r.RenderToWriter(ctx, renderReq, newStreamWriter(w))
	if deps.Metrics != nil {
		deps.Metrics.RecordRender(renderReq.Route.Pattern, timeSince(renderStart))
	}
	if err != nil {
		logError(deps.Logger, "pola: render", "err", err)
	}

	// Fill cache from tee buffer.
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
	// Inject CSP nonce if the security headers middleware is active.
	if nonce, ok := req.Context().Value(core.NonceContextKey).(string); ok && nonce != "" {
		params.Nonce = nonce
	}
	// Inject CSRF token as <meta> tags if the CSRF middleware is active.
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
			params.Scripts = append(params.Scripts, fmt.Sprintf("self.%s=%s", globals.SSRData, encoded))
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

// logError logs an error if a logger is configured.
func logError(logger core.Logger, msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

// defaultMetadata returns the built-in metadata used when the application has
// not supplied its own.
func defaultMetadata() *core.Metadata {
	faviconURL := "/public/favicon.ico"
	return &core.Metadata{
		Title: core.Title{Default: "Pola"},
		Icons: &core.Icons{
			Icon: []core.Icon{{URL: faviconURL}},
		},
	}
}

// ── teeWriter ─────────────────────────────────────────────────────────────────

// teeWriter wraps an http.ResponseWriter and captures all written bytes
// into a buffer while passing through to the underlying writer.
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

// ── streamWriter ──────────────────────────────────────────────────────────────

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

// ── VM helpers ────────────────────────────────────────────────────────────────

// prepareVM acquires a VM from the pool, applies injectors, sets the request
// context, and marshals props. The caller must release the VM when done.
func (r *Renderer) prepareVM(ctx context.Context, req core.RenderRequest) (core.SSRRuntime, string, error) {
	vm, err := r.pool.Acquire()
	if err != nil {
		return nil, "", fmt.Errorf("react renderer: acquire VM: %w", err)
	}

	for _, inj := range req.Injectors {
		if err := inj.Inject(ctx, vm); err != nil {
			r.pool.Release(vm)
			return nil, "", fmt.Errorf("react renderer: inject %s: %w", inj.Name(), err)
		}
	}

	if err := vm.SetRequestContext(req.RequestContext); err != nil {
		r.pool.Release(vm)
		return nil, "", fmt.Errorf("react renderer: set context: %w", err)
	}

	propsJSON, err := json.Marshal(req.Props)
	if err != nil {
		r.pool.Release(vm)
		return nil, "", fmt.Errorf("react renderer: marshal props: %w", err)
	}

	return vm, string(propsJSON), nil
}

// RenderToWriter acquires a VM, performs
// a full RSC Flight render, and streams all chunks to w.
func (r *Renderer) RenderToWriter(ctx context.Context, req core.RenderRequest, w core.StreamWriter) error {
	if r.pool == nil {
		return fmt.Errorf("react renderer: VM pool not configured")
	}

	vm, propsJSON, err := r.prepareVM(ctx, req)
	if err != nil {
		return err
	}
	defer r.pool.Release(vm)

	handle, err := vm.CallRenderFunction(req.Route.Export, propsJSON)
	if err != nil {
		return fmt.Errorf("react renderer: call render: %w", err)
	}

	wroteAny, err := vm.DrainStream(handle, w)
	if err != nil {
		return fmt.Errorf("react renderer: drain stream: %w", err)
	}
	if !wroteAny {
		return fmt.Errorf("react renderer: no Flight output written")
	}
	return nil
}

// ── bufferStreamWriter ────────────────────────────────────────────────────────

// bufferStreamWriter implements core.StreamWriter by writing to a bytes.Buffer.
type bufferStreamWriter struct{ buf *bytes.Buffer }

func (bw *bufferStreamWriter) WriteRaw(p []byte) (int, error) { return bw.buf.Write(p) }
func (bw *bufferStreamWriter) Flush()                         {}

// RenderToBytes performs a full RSC Flight render and returns the output as bytes.
// Used to capture Flight data for inline embedding in the HTML shell.
func (r *Renderer) RenderToBytes(ctx context.Context, req core.RenderRequest) ([]byte, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("react renderer: VM pool not configured")
	}

	vm, propsJSON, err := r.prepareVM(ctx, req)
	if err != nil {
		return nil, err
	}
	defer r.pool.Release(vm)

	handle, err := vm.CallRenderFunction(req.Route.Export, propsJSON)
	if err != nil {
		return nil, fmt.Errorf("react renderer: call render: %w", err)
	}

	var buf bytes.Buffer
	wroteAny, err := vm.DrainStream(handle, &bufferStreamWriter{buf: &buf})
	if err != nil {
		return nil, fmt.Errorf("react renderer: drain stream: %w", err)
	}
	if !wroteAny {
		return nil, fmt.Errorf("react renderer: no Flight output written")
	}
	return buf.Bytes(), nil
}

// WithPool returns a copy of the renderer that uses pool for VM acquisition.
func (r *Renderer) WithPool(pool core.SSRPool) *Renderer {
	nr := New()
	nr.pool = pool
	if d := r.deps.Load(); d != nil {
		cp := *d
		nr.deps.Store(&cp)
	}
	return nr
}

// WithShell returns a copy of the renderer that uses shell for HTML rendering.
func (r *Renderer) WithShell(shell core.HTMLShell) *Renderer {
	nr := New()
	nr.pool = r.pool
	d := r.loadDeps()
	d.Shell = shell
	nr.deps.Store(&d)
	return nr
}

// GenerateEntry implements the internal entryGenerator interface.
// It converts a DiscoveryResult into the in-memory server-entry TypeScript
// source that the bundler compiles into the server VM bundle.
func (r *Renderer) GenerateEntry(discovery core.DiscoveryResult) (string, error) {
	gen := &EntryGenerator{}
	return gen.Generate(EntryGenConfig{
		Pages:                 discovery.Pages,
		AppDir:                discovery.AppDir,
		GlobalErrorPath:       discovery.GlobalError,
		GlobalNotFoundPath:    discovery.GlobalNotFound,
		RootLayoutReturnsHTML: discovery.RootLayoutReturnsHTML,
	})
}

// ExtractDocumentProps calls __extractShell__ on a VM to get the root
// layout's HTML, then parses it with golang.org/x/net/html to extract
// document-level properties. Returns nil when the root layout doesn't
// return <html> or extraction fails.
func (r *Renderer) ExtractDocumentProps() (*core.DocumentProps, error) {
	if r.pool == nil {
		return nil, nil
	}
	vm, err := r.pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("react renderer: acquire VM for shell extraction: %w", err)
	}
	defer r.pool.Release(vm)

	result, err := vm.Call(globals.ExtractShellFn)
	if err != nil {
		return nil, nil // __extractShell__ not defined or failed — not an error
	}
	htmlStr, ok := result.(string)
	if !ok || htmlStr == "" {
		return nil, nil
	}
	return shell.ExtractDocumentProps(htmlStr)
}

// BundleConditions returns the esbuild conditions for the React RSC server pass.
func (r *Renderer) BundleConditions() []string {
	return []string{"react-server", "browser", "module", "default"}
}

// ClientEntry returns the package specifier for the React client hydration entry.
func (r *Renderer) ClientEntry() string {
	return "@pola/react/client"
}

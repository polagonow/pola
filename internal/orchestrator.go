// Package internal contains the core wiring for Pola applications.
// This package is not intended for direct import by end users.
package internal

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/polagonow/pola/core"
)

// streamingRenderer is the optional interface for renderers that can stream
// output directly to an http.ResponseWriter (RSC Flight protocol).
type streamingRenderer interface {
	RenderToWriter(ctx context.Context, req core.RenderRequest, w core.StreamWriter) error
}

// rscContentType is the MIME type for RSC Flight wire format requests.
const rscContentType = "text/x-component"

// Orchestrator implements http.Handler and wires all Pola components together.
type Orchestrator struct {
	registry      *core.Registry
	routes        []core.Route
	shell         core.HTMLShell
	assets        core.AssetServer
	bundleOutput  *core.BundleOutput
	notFoundRoute *core.Route // GlobalNotFound export, or nil
	dev           bool
	devScript     string // hot-reload inline script, set in dev mode
}

// NewOrchestrator creates a new Orchestrator from the given registry and build artifacts.
func NewOrchestrator(reg *core.Registry, routes []core.Route, shell core.HTMLShell, assets core.AssetServer, bundleOutput *core.BundleOutput, notFoundRoute *core.Route, dev bool) *Orchestrator {
	return &Orchestrator{
		registry:      reg,
		routes:        routes,
		shell:         shell,
		assets:        assets,
		bundleOutput:  bundleOutput,
		notFoundRoute: notFoundRoute,
		dev:           dev,
	}
}

// ServeHTTP handles a single HTTP request.
func (o *Orchestrator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Serve static assets under /public/.
	if strings.HasPrefix(r.URL.Path, "/public/") {
		o.assets.Handler("/public/").ServeHTTP(w, r)
		return
	}

	// Apply middleware in order
	handler := http.Handler(http.HandlerFunc(o.handle))
	for i := len(o.registry.Middleware) - 1; i >= 0; i-- {
		handler = o.registry.Middleware[i].Wrap(handler)
	}

	// Record metrics
	rw := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
	handler.ServeHTTP(rw, r)

	route, _ := o.registry.Router.Resolve(r.Context(), r.URL.Path)
	routeName := "unknown"
	if route != nil {
		routeName = route.Pattern
	}
	o.registry.Metrics.RecordRequest(routeName, r.Method, rw.code, time.Since(start))
}

func (o *Orchestrator) handle(w http.ResponseWriter, r *http.Request) {
	ctx, span := o.registry.Tracer.StartSpan(r.Context(), "pola.handle")
	defer span.End()

	route, params := o.registry.Router.Resolve(ctx, r.URL.Path)
	if route == nil {
		isRSC := r.Header.Get("Content-Type") == rscContentType
		if isRSC && o.notFoundRoute != nil {
			w.Header().Set("Content-Type", rscContentType+"; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			req := core.RenderRequest{
				Route:          *o.notFoundRoute,
				Props:          map[string]any{"params": map[string]any{}, "searchParams": map[string]any{}},
				RequestContext: buildRequestContext(r),
				Injectors:      o.registry.Injectors,
			}
			o.serveRSCBody(ctx, req, w)
			return
		}
		// HTML 404: return the client shell so the browser can bootstrap and
		// fetch the GlobalNotFound component via a subsequent RSC request.
		w.WriteHeader(http.StatusNotFound)
		o.serveHTML(w, r)
		return
	}

	req := core.RenderRequest{
		Route:          *route,
		Props:          buildPageProps(r, params),
		RequestContext: buildRequestContext(r),
		Injectors:      o.registry.Injectors,
	}

	if r.Header.Get("Content-Type") == rscContentType {
		o.serveRSC(w, r, ctx, req)
		return
	}
	o.serveHTML(w, r)
}

// serveRSC streams the RSC Flight wire format to the client.
func (o *Orchestrator) serveRSC(w http.ResponseWriter, r *http.Request, ctx context.Context, req core.RenderRequest) {
	// Set Content-Type before any WriteHeader call so it's included in the
	// response. If WriteHeader was already called (e.g. 404 not-found path),
	// Set is a no-op but that's fine — the header was already flushed.
	w.Header().Set("Content-Type", rscContentType+"; charset=utf-8")
	o.serveRSCBody(ctx, req, w)
}

// serveRSCBody writes the RSC Flight body to w. Headers must be set before calling.
func (o *Orchestrator) serveRSCBody(ctx context.Context, req core.RenderRequest, w http.ResponseWriter) {
	sr, ok := o.registry.Renderer.(streamingRenderer)
	if !ok {
		// Fallback: call Render and write body.
		result, err := o.registry.Renderer.Render(ctx, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(result.Body) //nolint:errcheck
		return
	}
	sw := &httpStreamWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		sw.flusher = f
	}
	if err := sr.RenderToWriter(ctx, req, sw); err != nil {
		// Headers already sent — log only.
		o.registry.Logger.Error("pola: RSC render", "err", err)
	}
}

// serveHTML returns the HTML shell for initial page loads.
// The client-side JS (Client.tsx) then fetches RSC Flight data separately.
func (o *Orchestrator) serveHTML(w http.ResponseWriter, _ *http.Request) {
	params := core.ShellParams{}
	if o.bundleOutput != nil {
		params.ImportURLs = o.bundleOutput.ImportURLs
		params.ClientScript = o.bundleOutput.ClientEntryURL
	}
	if o.devScript != "" {
		params.Scripts = append(params.Scripts, o.devScript)
	}
	html := o.shell.Render(params)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html)) //nolint:errcheck
}

// ── httpStreamWriter ──────────────────────────────────────────────────────────

type httpStreamWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (sw *httpStreamWriter) WriteRaw(p []byte) (int, error) { return sw.w.Write(p) }
func (sw *httpStreamWriter) Flush() {
	if sw.flusher != nil {
		sw.flusher.Flush()
	}
}

type responseRecorder struct {
	http.ResponseWriter
	code int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.code = code
	rr.ResponseWriter.WriteHeader(code)
}

func buildPageProps(r *http.Request, params map[string]any) map[string]any {
	searchParams := map[string]any{}
	for k, vs := range r.URL.Query() {
		if len(vs) == 1 {
			searchParams[k] = vs[0]
		} else {
			searchParams[k] = vs
		}
	}
	return map[string]any{"params": params, "searchParams": searchParams}
}

func buildRequestContext(r *http.Request) map[string]any {
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	return map[string]any{
		"url":     r.URL.String(),
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"method":  r.Method,
		"headers": headers,
	}
}

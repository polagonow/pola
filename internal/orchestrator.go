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

// requestHandler is an optional interface for renderers that want to own
// the full HTTP response for certain requests (e.g. RSC Flight streaming).
// Return handled=false to fall back to the HTML shell.
type requestHandler interface {
	ServeRequest(ctx context.Context, w http.ResponseWriter, r *http.Request,
		req core.RenderRequest, status int) (bool, error)
}

// Orchestrator implements http.Handler and wires all Pola components together.
type Orchestrator struct {
	registry      *core.Registry
	routes        []core.Route
	shell         core.HTMLShell
	assets        core.AssetServer
	bundleOutput  *core.BundleOutput
	notFoundRoute *core.Route // GlobalNotFound export, or nil
	dev           bool
	devScript     string       // hot-reload inline script, set in dev mode
	handler       http.Handler // middleware chain wrapping handle, built once
}

// NewOrchestrator creates a new Orchestrator from the given registry and build artifacts.
func NewOrchestrator(reg *core.Registry, routes []core.Route, shell core.HTMLShell, assets core.AssetServer, bundleOutput *core.BundleOutput, notFoundRoute *core.Route, dev bool) *Orchestrator {
	o := &Orchestrator{
		registry:      reg,
		routes:        routes,
		shell:         shell,
		assets:        assets,
		bundleOutput:  bundleOutput,
		notFoundRoute: notFoundRoute,
		dev:           dev,
	}
	if dev {
		o.devScript = ClientScript
	}
	// Build the middleware chain once at construction time instead of per request.
	handler := http.Handler(http.HandlerFunc(o.handle))
	for i := len(reg.Middleware) - 1; i >= 0; i-- {
		handler = reg.Middleware[i].Wrap(handler)
	}
	o.handler = handler
	return o
}

// ServeHTTP handles a single HTTP request.
func (o *Orchestrator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Serve static assets under /public/.
	if strings.HasPrefix(r.URL.Path, "/public/") {
		o.assets.Handler("/public/").ServeHTTP(w, r)
		return
	}

	// Metrics endpoint.
	if r.URL.Path == o.registry.Metrics.Path() {
		o.registry.Metrics.Handler().ServeHTTP(w, r)
		return
	}

	// Pprof endpoints (nil when not registered).
	if o.registry.Pprof != nil && strings.HasPrefix(r.URL.Path, o.registry.Pprof.Path()) {
		o.registry.Pprof.Handler().ServeHTTP(w, r)
		return
	}

	// Record metrics
	rw := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
	o.handler.ServeHTTP(rw, r)

	route, _ := o.registry.Router.Resolve(r.Context(), r.URL.Path)
	routeName := "unknown"
	if route != nil {
		routeName = route.Pattern
	}
	o.registry.Metrics.RecordRequest(routeName, r.Method, rw.code, time.Since(start))
}

// tryRendererServe calls ServeRequest on the renderer if it implements
// requestHandler. Returns true if the renderer claimed the response.
func (o *Orchestrator) tryRendererServe(ctx context.Context, w http.ResponseWriter, r *http.Request, req core.RenderRequest, status int) bool {
	rh, ok := o.registry.Renderer.(requestHandler)
	if !ok {
		return false
	}
	ctx, span := o.registry.Tracer.StartSpan(ctx, "pola.render")
	renderStart := time.Now()
	handled, err := rh.ServeRequest(ctx, w, r, req, status)
	span.End()
	o.registry.Metrics.RecordRender(req.Route.Pattern, time.Since(renderStart))
	if err != nil {
		o.registry.Logger.Error("pola: render", "err", err)
	}
	return handled
}

func (o *Orchestrator) handle(w http.ResponseWriter, r *http.Request) {
	ctx, span := o.registry.Tracer.StartSpan(r.Context(), "pola.handle")
	defer span.End()

	ctx, routeSpan := o.registry.Tracer.StartSpan(ctx, "pola.route.resolve")
	route, params := o.registry.Router.Resolve(ctx, r.URL.Path)
	routeSpan.End()

	if route == nil {
		if o.notFoundRoute != nil {
			req := core.RenderRequest{
				Route:          *o.notFoundRoute,
				Props:          map[string]any{"params": map[string]any{}, "searchParams": map[string]any{}},
				RequestContext: buildRequestContext(r),
				Injectors:      o.registry.Injectors,
			}
			if o.tryRendererServe(ctx, w, r, req, http.StatusNotFound) {
				return
			}
		}
		// HTML 404: client shell bootstraps, then fetches the not-found component.
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
	if o.tryRendererServe(ctx, w, r, req, http.StatusOK) {
		return
	}
	o.serveHTML(w, r)
}

// serveHTML returns the HTML shell for initial page loads.
// The client-side JS (Client.tsx) then fetches RSC Flight data separately.
func (o *Orchestrator) serveHTML(w http.ResponseWriter, _ *http.Request) {
	params := core.ShellParams{
		Metadata: defaultMetadata(),
	}
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

// defaultMetadata returns the built-in metadata used when the application has
// not supplied its own. It preserves the same title and favicon that were
// previously hardcoded in the HTML template.
func defaultMetadata() *core.Metadata {
	faviconURL := "/public/favicon.ico"
	return &core.Metadata{
		Title: core.Title{Default: "Pola"},
		Icons: &core.Icons{
			Icon: []core.Icon{{URL: faviconURL}},
		},
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

func (rr *responseRecorder) Flush() {
	if f, ok := rr.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
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

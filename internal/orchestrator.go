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

// contextKey is an unexported type for context keys in this package.
type contextKey int

const (
	routePatternKey contextKey = iota
	apiParamsKey
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
	renderer      core.Renderer
	router        core.Router
	apiRouter     core.APIRouter // may be nil; Go API route handlers
	logger        core.Logger
	metrics       core.Metrics
	tracer        core.Tracer
	pprof         core.Pprof // may be nil
	middleware    []core.Middleware
	injectors     []core.RuntimeInjector
	routes        []core.Route
	shell         core.HTMLShell
	assets        core.AssetServer
	bundleOutput  *core.BundleOutput
	notFoundRoute *core.Route        // GlobalNotFound export, or nil
	cssURLs       []string           // external stylesheet URLs
	documentProps *core.DocumentProps // extracted from root layout, or nil
	dev           bool
	devScript     string       // hot-reload inline script, set in dev mode
	handler       http.Handler // middleware chain wrapping handle, built once
}

// NewOrchestrator creates a new Orchestrator from resolved services and build artifacts.
func NewOrchestrator(
	renderer core.Renderer,
	router core.Router,
	apiRouter core.APIRouter,
	logger core.Logger,
	metrics core.Metrics,
	tracer core.Tracer,
	pprof core.Pprof,
	middleware []core.Middleware,
	injectors []core.RuntimeInjector,
	routes []core.Route,
	shell core.HTMLShell,
	assets core.AssetServer,
	bundleOutput *core.BundleOutput,
	notFoundRoute *core.Route,
	cssURLs []string,
	documentProps *core.DocumentProps,
	dev bool,
) *Orchestrator {
	o := &Orchestrator{
		renderer:      renderer,
		router:        router,
		apiRouter:     apiRouter,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
		pprof:         pprof,
		middleware:    middleware,
		injectors:     injectors,
		routes:        routes,
		shell:         shell,
		assets:        assets,
		bundleOutput:  bundleOutput,
		notFoundRoute: notFoundRoute,
		cssURLs:       cssURLs,
		documentProps: documentProps,
		dev:           dev,
	}
	if dev {
		o.devScript = ClientScript
	}
	// Build the middleware chain once at construction time instead of per request.
	handler := http.Handler(http.HandlerFunc(o.handle))
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i].Wrap(handler)
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

	// Metrics endpoint (nil when not registered).
	if o.metrics != nil && r.URL.Path == o.metrics.Path() {
		o.metrics.Handler().ServeHTTP(w, r)
		return
	}

	// Pprof endpoints (nil when not registered).
	if o.pprof != nil && strings.HasPrefix(r.URL.Path, o.pprof.Path()) {
		o.pprof.Handler().ServeHTTP(w, r)
		return
	}

	// Record metrics
	rw := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
	o.handler.ServeHTTP(rw, r)

	if o.metrics != nil {
		routeName := "unknown"
		if pat, ok := r.Context().Value(routePatternKey).(string); ok && pat != "" {
			routeName = pat
		}
		o.metrics.RecordRequest(routeName, r.Method, rw.code, time.Since(start))
	}
}

// tryRendererServe calls ServeRequest on the renderer if it implements
// requestHandler. Returns true if the renderer claimed the response.
func (o *Orchestrator) tryRendererServe(ctx context.Context, w http.ResponseWriter, r *http.Request, req core.RenderRequest, status int) bool {
	rh, ok := o.renderer.(requestHandler)
	if !ok {
		return false
	}
	if o.tracer != nil {
		var span core.Span
		ctx, span = o.tracer.StartSpan(ctx, "pola.render")
		defer span.End()
	}
	renderStart := time.Now()
	handled, err := rh.ServeRequest(ctx, w, r, req, status)
	if o.metrics != nil {
		o.metrics.RecordRender(req.Route.Pattern, time.Since(renderStart))
	}
	if err != nil {
		o.logger.Error("pola: render", "err", err)
	}
	return handled
}

func (o *Orchestrator) handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if o.tracer != nil {
		var span core.Span
		ctx, span = o.tracer.StartSpan(ctx, "pola.handle")
		defer span.End()
	}

	var routeSpan core.Span
	if o.tracer != nil {
		ctx, routeSpan = o.tracer.StartSpan(ctx, "pola.route.resolve")
	}
	route, params := o.router.Resolve(ctx, r.URL.Path)
	if routeSpan != nil {
		routeSpan.End()
	}

	// Store resolved route pattern for metrics (avoids re-resolving in ServeHTTP).
	if route != nil {
		ctx = context.WithValue(ctx, routePatternKey, route.Pattern)
	}
	*r = *r.WithContext(ctx)

	// API route integration: pages always win for GET requests.
	// For non-GET requests, or GET requests without a matching page, try API routes.
	isPageGET := route != nil && r.Method == http.MethodGet
	if !isPageGET && o.apiRouter != nil {
		if handler, apiParams, ok := o.apiRouter.Match(r); ok {
			if apiParams != nil {
				ctx = context.WithValue(ctx, apiParamsKey, apiParams)
				*r = *r.WithContext(ctx)
			}
			handler(w, r)
			return
		}
	}

	if route == nil {
		if o.notFoundRoute != nil {
			req := core.RenderRequest{
				Route:          *o.notFoundRoute,
				Props:          map[string]any{"params": map[string]any{}, "searchParams": map[string]any{}},
				RequestContext: buildRequestContext(r),
				Injectors:      o.injectors,
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
		Injectors:      o.injectors,
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
		Metadata:      defaultMetadata(),
		DocumentProps: o.documentProps,
	}
	if o.bundleOutput != nil {
		params.ImportURLs = o.bundleOutput.ImportURLs
		params.ClientScript = o.bundleOutput.ClientEntryURL
	}
	if len(o.cssURLs) > 0 {
		params.Stylesheets = o.cssURLs
	}
	if o.devScript != "" {
		params.Scripts = append(params.Scripts, o.devScript)
	}
	html := o.shell.Render(params)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html)) //nolint:errcheck
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

// allowedHeaders lists the HTTP headers that are safe to expose to JS server
// components.
var allowedHeaders = map[string]struct{}{
	"Accept":            {},
	"Accept-Encoding":   {},
	"Accept-Language":   {},
	"Cache-Control":     {},
	"Content-Language":  {},
	"Content-Type":      {},
	"Host":              {},
	"If-Modified-Since": {},
	"If-None-Match":     {},
	"Range":             {},
	"Referer":           {},
	"User-Agent":        {},
}

func buildRequestContext(r *http.Request) map[string]any {
	headers := make(map[string]string)
	for k, v := range r.Header {
		if _, ok := allowedHeaders[k]; ok {
			headers[k] = strings.Join(v, ", ")
		}
	}
	return map[string]any{
		"url":     r.URL.String(),
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"method":  r.Method,
		"headers": headers,
	}
}

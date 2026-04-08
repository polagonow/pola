// Package internal contains the core wiring for Pola applications.
// This package is not intended for direct import by end users.
package internal

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/routes"
)

// contextKey is an unexported type for context keys in this package.
type contextKey int

const (
	routePatternKey contextKey = iota
)

// Orchestrator implements http.Handler and wires all Pola components together.
type Orchestrator struct {
	renderer   core.Renderer
	router     core.Router
	apiRouter  core.APIRouter // may be nil; Go API route handlers
	logger     core.Logger
	metrics    core.Metrics
	tracer     core.Tracer
	pprof      core.Pprof  // may be nil
	cache      core.Cache  // may be nil; SSR cache for invalidation after mutations
	middleware []core.Middleware
	injectors  []core.RuntimeInjector
	routes     []core.Route
	assets     core.AssetServer
	dev        bool
	handler    http.Handler // middleware chain wrapping handle, built once
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
	cache core.Cache,
	middleware []core.Middleware,
	injectors []core.RuntimeInjector,
	routes []core.Route,
	assets core.AssetServer,
	dev bool,
) *Orchestrator {
	o := &Orchestrator{
		renderer:   renderer,
		router:     router,
		apiRouter:  apiRouter,
		logger:     logger,
		metrics:    metrics,
		tracer:     tracer,
		pprof:      pprof,
		cache:      cache,
		middleware: middleware,
		injectors:  injectors,
		routes:     routes,
		assets:     assets,
		dev:        dev,
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

func (o *Orchestrator) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Vary", "Content-Type")
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

	// API route integration: pages always win for GET requests.
	// For non-GET requests, or GET requests without a matching page, try API routes.
	isPageGET := route != nil && r.Method == http.MethodGet
	if !isPageGET && o.apiRouter != nil {
		if handler, apiParams, ok := o.apiRouter.Match(r); ok {
			req := r.WithContext(ctx)
			if apiParams != nil {
				req = routes.WithParams(req, apiParams)
			}
			// Wrap response to capture status for cache invalidation.
			rec := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
			handler(rec, req)
			// Invalidate SSR cache after successful mutations so pages
			// reflect the latest data without requiring a server restart.
			if o.cache != nil && slices.Contains(mutationMethods, r.Method) && rec.code >= 200 && rec.code < 300 {
				o.cache.Invalidate(ctx, "ssr:"+resourcePrefix(r.URL.Path))
			}
			return
		}
	}

	// Build per-request render context and delegate to the renderer.
	status := http.StatusOK
	var props map[string]any
	if route == nil {
		status = http.StatusNotFound
	} else {
		props = buildPageProps(r, params)
	}
	ctx = core.WithRenderRequest(ctx, route, props, buildRequestContext(r), status, o.injectors)
	o.renderer.ServeHTTP(w, r.WithContext(ctx))
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

// mutationMethods are HTTP methods that modify server state.
var mutationMethods = []string{"POST", "PUT", "PATCH", "DELETE"}

// resourcePrefix extracts the top-level resource path from a URL.
// e.g. "/articles/1/edit" → "/articles", "/articles" → "/articles".
func resourcePrefix(path string) string {
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) > 2 {
		return "/" + parts[1]
	}
	return path
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

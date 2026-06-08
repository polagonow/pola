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
	"github.com/polagonow/pola/core/reserved"
	"github.com/polagonow/pola/routes"
	"github.com/polagonow/pola/serveraction"
)

// Server-action endpoint paths. These bypass the page/API router and the
// page-oriented middleware chain; the handlers perform their own CSRF check.
const (
	serverActionPath = "/_pola/action"
	formActionPath   = "/_pola/form-action"
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
	pprof      core.Pprof // may be nil
	cache      core.Cache // may be nil; SSR cache for invalidation after mutations
	middleware []core.Middleware
	injectors  []core.RuntimeInjector
	routes     []core.Route
	assets     core.AssetServer
	dev        bool
	handler    http.Handler // middleware chain wrapping handle, built once

	// Server-action handlers (nil when the renderer/bundle has no actions).
	actionHandler     http.HandlerFunc
	formActionHandler http.HandlerFunc
}

// SetServerActionHandlers installs the /_pola/action and /_pola/form-action
// handlers. A nil argument leaves the endpoints unserved (404 via normal
// routing). Called by the pipeline after the orchestrator is constructed.
func (o *Orchestrator) SetServerActionHandlers(h *serveraction.Handler) {
	if h == nil {
		o.actionHandler, o.formActionHandler = nil, nil
		return
	}
	o.actionHandler = h.Action
	o.formActionHandler = h.Form
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

	// Serve framework-built JS/CSS chunks under the reserved /_pola/assets/
	// namespace. These live in <publicDir>/assets, so stripping the /_pola
	// prefix maps the request onto that subdirectory.
	if strings.HasPrefix(r.URL.Path, reserved.Assets+"/") {
		o.assets.Handler(reserved.Prefix).ServeHTTP(w, r)
		return
	}

	// Serve the user's static files under /public/.
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

	// Server-action endpoints. These bypass the page/API router and the
	// page-oriented middleware chain; the handler performs its own CSRF check.
	if o.actionHandler != nil && r.URL.Path == serverActionPath {
		o.actionHandler(w, r)
		return
	}
	if o.formActionHandler != nil && r.URL.Path == formActionPath {
		o.formActionHandler(w, r)
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

// newServerActionHandler builds the server-action handler when the renderer
// supports server actions and the bundle declares at least one. Returns nil
// otherwise. The returned handler is installed on the orchestrator via
// SetServerActionHandlers.
func newServerActionHandler(
	renderer core.Renderer,
	bundleOutput *core.BundleOutput,
	injectors []core.RuntimeInjector,
	cache core.Cache,
	dev bool,
	logger core.Logger,
) *serveraction.Handler {
	inv, ok := renderer.(core.ServerActionInvoker)
	if !ok || bundleOutput == nil || len(bundleOutput.ServerActions) == 0 {
		return nil
	}
	valid := make(map[string]struct{}, len(bundleOutput.ServerActions))
	for id := range bundleOutput.ServerActions {
		valid[id] = struct{}{}
	}
	h := &serveraction.Handler{
		Invoker:      inv,
		ValidIDs:     valid,
		BuildContext: buildRequestContext,
		Injectors:    injectors,
		Dev:          dev,
		Logger:       logger,
	}
	if cache != nil {
		h.Invalidate = func(ctx context.Context, path string) {
			_ = cache.Invalidate(ctx, "ssr:"+resourcePrefix(path))
		}
	}
	return h
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

// Package internal contains the core wiring for Pola applications.
// This package is not intended for direct import by end users.
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
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
	cache         core.Cache // may be nil; render cache
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
	cache core.Cache,
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
		cache:         cache,
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
//
// For Flight requests (Content-Type: text/x-component), it checks the cache
// first and serves cached data directly. On cache miss with Route.Revalidate > 0,
// it tees the response to fill the cache for subsequent requests.
func (o *Orchestrator) tryRendererServe(ctx context.Context, w http.ResponseWriter, r *http.Request, req core.RenderRequest, status int) bool {
	rh, ok := o.renderer.(requestHandler)
	if !ok {
		return false
	}

	// Cache integration for Flight requests.
	isFlight := r.Header.Get("Content-Type") == "text/x-component"
	if isFlight && o.cache != nil {
		cacheKey := "ssr:" + req.Route.Pattern + "?" + r.URL.RawQuery
		// Serve from cache if available — instant, no VM needed.
		if cached, ok, err := o.cache.Get(ctx, cacheKey); err != nil {
			o.logger.Error("pola: cache get", "key", cacheKey, "err", err)
		} else if ok {
			w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
			w.WriteHeader(status)
			w.Write(cached) //nolint:errcheck
			return true
		}
		// On cache miss, tee the output to fill cache for next request.
		tw := &teeWriter{ResponseWriter: w}
		defer func() {
			if len(tw.buf) > 0 {
				if err := o.cache.Set(ctx, cacheKey, tw.buf, core.CacheOptions{
					TTL: req.Route.Revalidate, // 0 = no expiry
				}); err != nil {
					o.logger.Error("pola: cache set", "key", cacheKey, "err", err)
				}
			}
		}()
		w = tw
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
		w.WriteHeader(http.StatusNotFound)
		o.serveHTML(w, r, nil)
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

	// If cached Flight data exists, embed it in the HTML shell to avoid
	// a second request (Next.js-style inline SSR data).
	var ssrData []byte
	if o.cache != nil {
		cacheKey := "ssr:" + route.Pattern + "?" + r.URL.RawQuery
		if cached, ok, err := o.cache.Get(ctx, cacheKey); err != nil {
			o.logger.Error("pola: cache get", "key", cacheKey, "err", err)
		} else if ok {
			ssrData = cached
		}
	}
	o.serveHTML(w, r, ssrData)
}

// serveHTML returns the HTML shell for initial page loads.
// When ssrData is non-nil, it is embedded as an inline script so the client
// can use it directly without a second Flight request.
func (o *Orchestrator) serveHTML(w http.ResponseWriter, _ *http.Request, ssrData []byte) {
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
	if len(ssrData) > 0 {
		// json.Marshal escapes <, >, & to \uXXXX — safe inside <script> tags.
		encoded, err := json.Marshal(string(ssrData))
		if err != nil {
			o.logger.Error("pola: marshal SSR data", "err", err)
		} else {
			params.Scripts = append(params.Scripts, fmt.Sprintf("self.%s=%s", globals.SSRData, encoded))
		}
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
		if _, ok := allowedHeaders[http.CanonicalHeaderKey(k)]; ok {
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

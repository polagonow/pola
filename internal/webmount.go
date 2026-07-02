package internal

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/reserved"
	"github.com/polagonow/pola/routes"
	"github.com/polagonow/pola/serveraction"
	"github.com/polagonow/pola/webframework/std"
)

// Server-action endpoint paths. These are mounted as plain http.Handlers; the
// handlers perform their own CSRF check.
const (
	serverActionPath = "/_pola/action"
	formActionPath   = "/_pola/form-action"
)

// newFramework resolves a fresh WebFramework from the registered factory,
// defaulting to the std (net/http) adapter when none is registered. A factory
// (not a singleton) is used so each (re)mount gets a clean engine — dev
// hot-reload rebuilds the handler without re-registering routes onto an engine
// that already has them.
func newFramework(registry *core.Registry) core.WebFramework {
	if factory, err := core.Invoke[core.WebFrameworkFactory](registry); err == nil && factory != nil {
		return factory()
	}
	return std.New()
}

// getRouteSpecs discovers route specs once and caches them in the registry so
// later (re)mounts replay the cached specs instead of draining the now-empty
// registration queues.
func getRouteSpecs(registry *core.Registry) (routes.RouteSpecs, error) {
	if specs, err := core.Invoke[routes.RouteSpecs](registry); err == nil {
		return specs, nil
	}
	specs, err := routes.Discover(registry)
	if err != nil {
		return nil, err
	}
	core.ProvideValue[routes.RouteSpecs](registry, specs)
	return specs, nil
}

// mountDeps holds everything mountApp needs to wire a WebFramework.
type mountDeps struct {
	fw         core.WebFramework
	renderer   core.Renderer // nil → api-only (404 fallback)
	router     core.Router   // nil → api-only
	cache      core.Cache    // may be nil
	metrics    core.Metrics  // may be nil
	tracer     core.Tracer   // may be nil
	pprof      core.Pprof    // may be nil
	assets     core.AssetServer
	action     *serveraction.Handler // may be nil
	middleware []core.Middleware
	injectors  []core.RuntimeInjector
	specs      routes.RouteSpecs
}

// mountApp wires Pola's machinery onto the selected WebFramework and returns its
// root http.Handler. It replaces the historical bespoke Orchestrator: reserved
// endpoints are mounted, API routes registered (with page precedence for GET,
// SSR cache invalidation for mutations, and metrics), middleware applied, and
// the page renderer set as the catch-all fallback.
func mountApp(d mountDeps) http.Handler {
	fw := d.fw

	// Page-oriented middleware chain (applied around the whole engine).
	for _, mw := range d.middleware {
		fw.Use(mw.Wrap)
	}

	// Reserved mounts (all plain http.Handlers).
	if d.assets != nil {
		fw.Mount(reserved.Assets, d.assets.Handler(reserved.Prefix))
		fw.Mount("/public/", d.assets.Handler("/public/"))
	}
	if d.metrics != nil {
		fw.Mount(d.metrics.Path(), d.metrics.Handler())
	}
	if d.pprof != nil {
		fw.Mount(d.pprof.Path(), d.pprof.Handler())
	}
	if d.action != nil {
		fw.Mount(serverActionPath, http.HandlerFunc(d.action.Action))
		fw.Mount(formActionPath, http.HandlerFunc(d.action.Form))
	}

	// renderRoute renders a resolved page route (or a 404 when route is nil).
	renderRoute := func(w http.ResponseWriter, r *http.Request, route *core.Route, params map[string]any) {
		w.Header().Set("Vary", "Content-Type")
		ctx := r.Context()
		if d.tracer != nil {
			var span core.Span
			ctx, span = d.tracer.StartSpan(ctx, "pola.render")
			defer span.End()
		}
		status := http.StatusOK
		var props map[string]any
		if route == nil {
			status = http.StatusNotFound
		} else {
			props = buildPageProps(r, params)
		}
		ctx = core.WithRenderRequest(ctx, route, props, buildRequestContext(r), status, d.injectors)
		rec := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		d.renderer.ServeHTTP(rec, r.WithContext(ctx))
		if d.metrics != nil {
			name := "unknown"
			if route != nil {
				name = route.Pattern
			}
			d.metrics.RecordRequest(name, r.Method, rec.code, time.Since(start))
		}
	}

	// Register API routes.
	for _, spec := range d.specs {
		method, pattern := spec.Method, spec.Pattern
		h := spec.Handler

		// SSR cache invalidation after a successful mutation.
		if d.cache != nil && isMutationMethod(method) {
			inner := h
			h = func(c core.Context) error {
				err := inner(c)
				if s := c.Status(); s >= 200 && s < 300 {
					_ = d.cache.Invalidate(c.Ctx(), "ssr:"+resourcePrefix(c.Request().URL.Path))
				}
				return err
			}
		}

		// Per-route metrics.
		if d.metrics != nil {
			inner := h
			m, pat := method, pattern
			h = func(c core.Context) error {
				start := time.Now()
				err := inner(c)
				d.metrics.RecordRequest(pat, m, c.Status(), time.Since(start))
				return err
			}
		}

		// Pages always win for GET: if the page router resolves the path,
		// render it instead of running the API handler.
		if method == http.MethodGet && d.renderer != nil && d.router != nil {
			api := h
			h = func(c core.Context) error {
				r := c.Request()
				if route, params := d.router.Resolve(r.Context(), r.URL.Path); route != nil {
					renderRoute(c.Writer(), r, route, params)
					return nil
				}
				return api(c)
			}
		}

		fw.Handle(method, pattern, h)
	}

	// Fallback: page renderer (full-stack) or 404 (api-only).
	if d.renderer != nil && d.router != nil {
		fw.Fallback(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, params := d.router.Resolve(r.Context(), r.URL.Path)
			renderRoute(w, r, route, params)
		}))
	} else {
		fw.Fallback(http.NotFoundHandler())
	}

	return fw.Handler()
}

func isMutationMethod(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// ── Helpers (moved from the retired orchestrator) ────────────────────────────

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

// allowedHeaders lists the HTTP headers safe to expose to JS server components.
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
// supports server actions and the bundle declares at least one; returns nil
// otherwise.
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

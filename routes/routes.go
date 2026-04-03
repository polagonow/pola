// Package routes provides Go-based API route handlers using livebud-style
// RESTful conventions. Route structs are registered via init() and discovered
// at startup through reflection.
//
// Routes and frontend pages share the same URL namespace. Frontend pages
// always win for GET requests; routes handle non-GET methods on the same path,
// or serve all methods when no frontend page exists.
package routes

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/patternmatch"
)

// ── Global registration (called from user init() functions) ─────────────────

var (
	mu      sync.Mutex
	pending []func(*core.Registry) any // route factories registered before Build()
)

// Register queues a route factory for discovery. The factory receives the DI
// registry and returns a fully-constructed route struct. Must be called from init().
func Register(factory func(*core.Registry) any) {
	mu.Lock()
	defer mu.Unlock()
	pending = append(pending, factory)
}

// drain returns and clears the pending registrations.
func drain() []func(*core.Registry) any {
	mu.Lock()
	defer mu.Unlock()
	out := pending
	pending = nil
	return out
}

// ── Router ──────────────────────────────────────────────────────────────────

// Router resolves HTTP requests to registered Go API route handlers.
type Router struct {
	mu            sync.RWMutex
	staticRoutes  map[string][]*apiRoute // keyed by pattern (static paths)
	dynamicRoutes []*apiRoute            // sorted by specificity
	sorted        bool
}

// apiRoute is a single compiled route entry (one per method+pattern combo).
type apiRoute struct {
	pattern  string
	method   string // HTTP method (GET, POST, etc.)
	compiled *patternmatch.CompiledRoute
	handler  http.HandlerFunc
}

// New returns a new empty Router.
func New() *Router {
	return &Router{
		staticRoutes: make(map[string][]*apiRoute),
	}
}

// Build drains all registered route factories, calls each with the registry
// to produce fully-constructed route structs, discovers RESTful methods via
// reflection, and compiles the routing table.
func (r *Router) Build(registry *core.Registry) error {
	factories := drain()
	if len(factories) == 0 {
		return nil
	}

	// Materialize all handlers by calling factories with the registry.
	handlers := make([]any, len(factories))
	for i, f := range factories {
		handlers[i] = f(registry)
	}

	// First pass: collect all package paths to know which are registered resources.
	registeredPkgs := make(map[string]bool, len(handlers))
	for _, h := range handlers {
		relPath := relativePackagePath(h)
		if relPath != "" {
			registeredPkgs[relPath] = true
		}
	}

	for _, h := range handlers {
		// Derive URL pattern: use Path() override if provided, else auto-derive.
		var basePattern string
		if p, ok := h.(Pather); ok {
			basePattern = p.Path()
			if !strings.HasPrefix(basePattern, "/") {
				typeName := reflect.TypeOf(h).Elem().Name()
				return fmt.Errorf("routes: %s.Path() must return a path starting with '/', got %q", typeName, basePattern)
			}
		} else {
			var memberOverride *bool
			if m, ok := h.(Memberer); ok {
				v := m.Member()
				memberOverride = &v
			}
			pkgPath := reflect.TypeOf(h).Elem().PkgPath()
			basePattern = pathToPattern(pkgPath, registeredPkgs, memberOverride)
		}

		// Discover RESTful actions on the struct.
		discovered, err := discoverActions(h)
		if err != nil {
			return err
		}

		compiled := patternmatch.CompilePattern(basePattern)
		for _, action := range discovered {
			ar := &apiRoute{
				pattern:  basePattern,
				method:   action.Method,
				compiled: compiled,
				handler:  action.Handler,
			}

			if compiled.IsStatic {
				r.staticRoutes[basePattern] = append(r.staticRoutes[basePattern], ar)
			} else {
				r.dynamicRoutes = append(r.dynamicRoutes, ar)
			}
		}
	}

	// Sort dynamic routes by specificity.
	sortAPIRoutes(r.dynamicRoutes)
	r.sorted = true

	return nil
}

// Match resolves the request URL and method to a handler and extracted params.
// Returns (nil, nil, false) if no route matches.
func (r *Router) Match(req *http.Request) (http.HandlerFunc, map[string]any, bool) {
	path := patternmatch.NormalizePath(req.URL.Path)
	method := req.Method

	r.mu.RLock()
	defer r.mu.RUnlock()

	// O(1) static lookup.
	if routes, ok := r.staticRoutes[path]; ok {
		for _, ar := range routes {
			if ar.method == method {
				return ar.handler, map[string]any{}, true
			}
		}
		// Pattern matched but method didn't — check dynamic routes too before 405.
	}

	// Linear scan of sorted dynamic routes.
	var methodMatched bool
	for _, ar := range r.dynamicRoutes {
		params, ok := patternmatch.Match(ar.compiled, path)
		if !ok {
			continue
		}
		if ar.method == method {
			return ar.handler, params, true
		}
		methodMatched = true
	}

	// If we matched a static pattern but not the method, check all static entries.
	if !methodMatched {
		if _, ok := r.staticRoutes[path]; ok {
			methodMatched = true
		}
	}

	// 405: pattern matched but method didn't.
	if methodMatched {
		allowed := r.allowedMethods(path)
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}, nil, true
	}

	return nil, nil, false
}

// allowedMethods collects all HTTP methods registered for a given path.
func (r *Router) allowedMethods(path string) []string {
	seen := map[string]bool{}

	if routes, ok := r.staticRoutes[path]; ok {
		for _, ar := range routes {
			seen[ar.method] = true
		}
	}

	for _, ar := range r.dynamicRoutes {
		if _, ok := patternmatch.Match(ar.compiled, path); ok {
			seen[ar.method] = true
		}
	}

	methods := make([]string, 0, len(seen))
	for m := range seen {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return methods
}

// relativePackagePath returns the package path relative to the "routes" segment.
// e.g. "myapp/routes/users/posts" → "users/posts", "myapp/routes" → "".
func relativePackagePath(h any) string {
	pkgPath := reflect.TypeOf(h).Elem().PkgPath()
	parts := strings.Split(pkgPath, "/")
	for i, p := range parts {
		if p == "routes" {
			rest := parts[i+1:]
			if len(rest) == 0 {
				return ""
			}
			return strings.Join(rest, "/")
		}
	}
	return ""
}

// sortAPIRoutes sorts dynamic API routes by segment specificity.
func sortAPIRoutes(routes []*apiRoute) {
	compiled := make([]*patternmatch.CompiledRoute, len(routes))
	for i, ar := range routes {
		compiled[i] = ar.compiled
	}

	sort.SliceStable(routes, func(i, j int) bool {
		a, b := routes[i].compiled.Segments, routes[j].compiled.Segments
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		for k := 0; k < minLen; k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		if len(a) != len(b) {
			return len(a) > len(b)
		}
		return routes[i].pattern < routes[j].pattern
	})
}

// WithParams returns a new request with route params injected into its context.
func WithParams(r *http.Request, params map[string]any) *http.Request {
	ctx := context.WithValue(r.Context(), paramsKey, params)
	return r.WithContext(ctx)
}

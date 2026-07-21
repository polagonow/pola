// Package std implements core.WebFramework on top of the Go net/http standard
// library. It is Pola's default web framework and reproduces the historical
// behavior of the (now retired) internal orchestrator: O(1) static API-route
// lookup, a radix tree for dynamic routes, 405 handling with an Allow header,
// reserved-prefix mounts via http.ServeMux, and a catch-all fallback.
package std

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/patternmatch"
)

// Framework is the net/http-backed core.WebFramework.
type Framework struct {
	mu sync.Mutex

	static  map[string][]stdRoute // pattern → routes (static paths)
	dynamic []stdRoute            // dynamic routes (with params)

	table    *patternmatch.RouteTable
	routeMap map[*patternmatch.CompiledRoute][]stdRoute

	mux      *http.ServeMux // reserved-prefix mounts
	fallback http.Handler
	mws      []func(http.Handler) http.Handler

	built sync.Once
}

// stdRoute is a single compiled API route (one per method+pattern).
type stdRoute struct {
	pattern  string
	method   string
	compiled *patternmatch.CompiledRoute
	handler  core.HandlerFunc
}

// New returns a new, empty std Framework.
func New() *Framework {
	return &Framework{
		static: make(map[string][]stdRoute),
		mux:    http.NewServeMux(),
	}
}

// Name implements core.WebFramework.
func (f *Framework) Name() string { return "std" }

// Handle implements core.WebFramework.
func (f *Framework) Handle(method, pattern string, h core.HandlerFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	compiled := patternmatch.CompilePattern(pattern)
	ar := stdRoute{pattern: pattern, method: method, compiled: compiled, handler: h}
	if compiled.IsStatic {
		f.static[pattern] = append(f.static[pattern], ar)
	} else {
		f.dynamic = append(f.dynamic, ar)
	}
}

// Use implements core.WebFramework.
func (f *Framework) Use(mw func(http.Handler) http.Handler) {
	f.mws = append(f.mws, mw)
}

// Mount implements core.WebFramework. A non-slash-terminated prefix is also
// registered with a trailing slash so subtree requests (assets, pprof, /public/)
// resolve to the same handler.
func (f *Framework) Mount(prefix string, h http.Handler) {
	f.mux.Handle(prefix, h)
	if !strings.HasSuffix(prefix, "/") {
		f.mux.Handle(prefix+"/", h)
	}
}

// Fallback implements core.WebFramework.
func (f *Framework) Fallback(h http.Handler) { f.fallback = h }

// Handler implements core.WebFramework: it finalizes the routing table and
// wraps the dispatcher with the registered middleware (in registration order).
func (f *Framework) Handler() http.Handler {
	f.build()
	var h http.Handler = http.HandlerFunc(f.serve)
	for i := len(f.mws) - 1; i >= 0; i-- {
		h = f.mws[i](h)
	}
	return h
}

// build compiles the dynamic radix tree once.
func (f *Framework) build() {
	f.built.Do(func() {
		sortRoutes(f.dynamic)
		f.routeMap = make(map[*patternmatch.CompiledRoute][]stdRoute)
		seen := make(map[string]*patternmatch.CompiledRoute)
		var compiled []*patternmatch.CompiledRoute
		for _, ar := range f.dynamic {
			if _, ok := seen[ar.pattern]; !ok {
				seen[ar.pattern] = ar.compiled
				compiled = append(compiled, ar.compiled)
			}
			f.routeMap[seen[ar.pattern]] = append(f.routeMap[seen[ar.pattern]], ar)
		}
		f.table = patternmatch.NewRouteTable(compiled)
	})
}

func (f *Framework) serve(w http.ResponseWriter, r *http.Request) {
	// 1. Reserved-prefix mounts (assets, /public/, metrics, pprof, actions).
	if h, pat := f.mux.Handler(r); pat != "" {
		h.ServeHTTP(w, r)
		return
	}

	// 2. API routes.
	if h, params, ok := f.match(r); ok {
		if params != nil {
			r = core.WithParams(r, params)
		}
		c := newContext(w, r)
		if err := h(c); err != nil && !c.w.wrote {
			// A handler that returns core.WithStatus(...) (or any error carrying
			// a StatusCoder — e.g. a 404 for a not-found) maps to that status;
			// everything else is a 500.
			_ = c.JSON(core.StatusOf(err), core.M{"error": err.Error()})
		}
		return
	}

	// 3. Fallback (page renderer or 404).
	if f.fallback != nil {
		f.fallback.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

// match resolves the request to a handler and params, returning a 405 handler
// when a pattern matches but the method does not.
func (f *Framework) match(r *http.Request) (core.HandlerFunc, map[string]any, bool) {
	path := patternmatch.NormalizePath(r.URL.Path)
	method := r.Method

	// O(1) static lookup.
	if rts, ok := f.static[path]; ok {
		for _, ar := range rts {
			if ar.method == method {
				return ar.handler, map[string]any{}, true
			}
		}
	}

	var methodMatched bool
	if f.table != nil {
		if compiled, params, ok := f.table.Match(path); ok {
			if handlers, found := f.routeMap[compiled]; found {
				for _, ar := range handlers {
					if ar.method == method {
						return ar.handler, params, true
					}
					methodMatched = true
				}
			}
		}
	}

	if !methodMatched {
		if _, ok := f.static[path]; ok {
			methodMatched = true
		}
	}

	if methodMatched {
		allowed := f.allowedMethods(path)
		return func(c core.Context) error {
			c.SetHeader("Allow", strings.Join(allowed, ", "))
			return c.String(http.StatusMethodNotAllowed, "Method Not Allowed")
		}, nil, true
	}

	return nil, nil, false
}

// allowedMethods collects all methods registered for a path (for the Allow header).
func (f *Framework) allowedMethods(path string) []string {
	seen := map[string]bool{}
	if rts, ok := f.static[path]; ok {
		for _, ar := range rts {
			seen[ar.method] = true
		}
	}
	if f.table != nil {
		if compiled, _, ok := f.table.Match(path); ok {
			for _, ar := range f.routeMap[compiled] {
				seen[ar.method] = true
			}
		}
	}
	methods := make([]string, 0, len(seen))
	for m := range seen {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return methods
}

// sortRoutes orders dynamic routes by segment specificity (ported verbatim from
// the original routes.Router so matching precedence is unchanged).
func sortRoutes(rts []stdRoute) {
	sort.SliceStable(rts, func(i, j int) bool {
		a, b := rts[i].compiled.Segments, rts[j].compiled.Segments
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
		return rts[i].pattern < rts[j].pattern
	})
}

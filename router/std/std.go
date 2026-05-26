// Package std provides a simple path-based router stub for the Pola framework.
// It implements core.Router with exact-match path routing (no dynamic segments).
package std

import (
	"context"
	"sync"

	"github.com/polagonow/pola/core"
)

// Router is a simple path-based router that matches URL paths by exact string
// equality. It does not support dynamic segments or catch-all patterns.
type Router struct {
	mu     sync.RWMutex
	routes map[string]core.Route // pattern → Route
}

// New returns a new std Router.
func New() *Router {
	return &Router{routes: make(map[string]core.Route)}
}

// Name returns the router name.
func (r *Router) Name() string { return "std" }

// ScanRoutes registers each discovered file as a static route. The URL pattern
// is derived by stripping the file extension and replacing OS path separators
// with forward slashes.
func (r *Router) ScanRoutes(_ context.Context, _ core.FS, _ string, _ []string) ([]core.Route, error) {
	// Stub: no file-system scanning implemented.
	// Users are expected to register routes manually or use the nextjs router.
	return nil, nil
}

// Resolve returns the route whose pattern exactly matches path.
func (r *Router) Resolve(_ context.Context, path string) (*core.Route, map[string]any) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if rt, ok := r.routes[path]; ok {
		return &rt, map[string]any{}
	}
	return nil, nil
}

// Register adds a route to the router.
func (r *Router) Register(route core.Route) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[route.Pattern] = route
}

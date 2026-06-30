// Package routes provides Go-based API route handlers
// RESTful conventions. Route structs (a struct with verb methods) and
// function-based routes (package-level functions named after HTTP verbs) are
// registered via init() and discovered at build time through reflection.
//
// Discovery (Discover) is framework-neutral: it produces RouteSpecs that the
// pipeline registers onto the selected core.WebFramework (std/gin/echo/chi).
// Matching and dispatch are owned by that framework, not this package.
//
// Routes and frontend pages share the same URL namespace. Frontend pages
// always win for GET requests; routes handle non-GET methods on the same path,
// or serve all methods when no frontend page exists.
package routes

import (
	"strings"
	"sync"

	"github.com/polagonow/pola/core"
)

// ── Global registration (called from user init() functions) ─────────────────

var (
	mu      sync.Mutex
	pending []func(*core.Registry) any // struct route factories registered before Discover()
)

// Register queues a route factory for discovery. The factory receives the DI
// registry and returns a fully-constructed route struct. Must be called from init().
func Register(factory func(*core.Registry) any) {
	mu.Lock()
	defer mu.Unlock()
	pending = append(pending, factory)
}

// drain returns and clears the pending struct registrations.
func drain() []func(*core.Registry) any {
	mu.Lock()
	defer mu.Unlock()
	out := pending
	pending = nil
	return out
}

// relativePackagePath returns the package path relative to the "routes" segment.
// e.g. "myapp/routes/users/posts" → "users/posts", "myapp/routes" → "".
func relativePackagePath(pkgPath string) string {
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

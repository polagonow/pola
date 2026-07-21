package routes

import (
	"testing"

	"github.com/polagonow/pola/core"
)

type protectedRoute struct{}

func (protectedRoute) Middleware() []core.RouteMiddleware {
	return []core.RouteMiddleware{func(n core.HandlerFunc) core.HandlerFunc { return n }}
}
func (protectedRoute) Meta() map[string]any { return map[string]any{"auth": true} }

func TestRouteMiddlewareMetaCollected(t *testing.T) {
	var r any = protectedRoute{}
	if got := routeMiddleware(r); len(got) != 1 {
		t.Errorf("routeMiddleware len = %d, want 1", len(got))
	}
	if got := routeMeta(r); got["auth"] != true {
		t.Errorf("routeMeta = %v, want auth:true", got)
	}

	// A plain route implementing neither interface yields nil for both.
	type plain struct{}
	if routeMiddleware(plain{}) != nil || routeMeta(plain{}) != nil {
		t.Error("plain route should have no middleware/meta")
	}
}

func TestSplitActionsAttachesExtras(t *testing.T) {
	mws := []core.RouteMiddleware{func(n core.HandlerFunc) core.HandlerFunc { return n }}
	meta := map[string]any{"auth": true}
	actions := []discoveredAction{
		{Method: "GET", Handler: func(core.Context) error { return nil }},
		{Method: "PUT", Handler: func(core.Context) error { return nil }},
	}

	specs := splitActions("/users", actions, mws, meta)
	if len(specs) != 2 {
		t.Fatalf("specs = %d, want 2", len(specs))
	}
	for _, s := range specs {
		if len(s.Middleware) != 1 {
			t.Errorf("%s: per-route middleware not attached", s.Method)
		}
		if s.Meta["auth"] != true {
			t.Errorf("%s: meta not attached", s.Method)
		}
		switch s.Method {
		case "GET":
			if s.Pattern != "/users" {
				t.Errorf("GET pattern = %q, want /users", s.Pattern)
			}
		case "PUT": // member method → /:id
			if s.Pattern != "/users/:id" {
				t.Errorf("PUT pattern = %q, want /users/:id", s.Pattern)
			}
		}
	}
}

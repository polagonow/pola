package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/webframework/std"
)

// ── Test route types ─────────────────────────────────────────────────────────

// usersRoute is a simple struct route.
type usersRoute struct {
	getBody  string
	postBody string
}

func (r *usersRoute) GET(c core.Context) error  { return c.String(200, r.getBody) }
func (r *usersRoute) POST(c core.Context) error { return c.String(201, r.postBody) }

// ctxRoute uses the framework-neutral func(core.Context) error signature.
type ctxRoute struct{}

func (*ctxRoute) GET(c core.Context) error { return c.String(200, "ctx-ok") }
func (*ctxRoute) DELETE(c core.Context) error {
	return c.String(200, "deleted:"+c.Param("id"))
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// mount builds a std framework from specs and returns its handler.
func mount(specs RouteSpecs) http.Handler {
	fw := std.New()
	for _, s := range specs {
		fw.Handle(s.Method, s.Pattern, s.Handler)
	}
	return fw.Handler()
}

func do(h http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

// specsFor discovers specs for a single struct without touching global queues.
func specsFor(t *testing.T, h any, registered map[string]bool) RouteSpecs {
	t.Helper()
	base, err := basePatternFor(h, registered)
	if err != nil {
		t.Fatalf("basePatternFor: %v", err)
	}
	actions, err := discoverActions(h)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}
	return splitActions(base, actions)
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestStructRoute(t *testing.T) {
	specs := specsFor(t, &usersRoute{getBody: "list", postBody: "created"}, nil)
	// usersRoute is defined in package "routes", so base pattern derives to "/".
	h := mount(specs)

	if w := do(h, "GET", "/"); w.Body.String() != "list" {
		t.Errorf("GET / = %q, want %q", w.Body.String(), "list")
	}
	if w := do(h, "POST", "/"); w.Body.String() != "created" {
		t.Errorf("POST / = %q, want %q", w.Body.String(), "created")
	}
	// PUT not defined → 405 with Allow header.
	w := do(h, "PUT", "/")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT / status = %d, want 405", w.Code)
	}
	if w.Header().Get("Allow") == "" {
		t.Error("expected Allow header on 405")
	}
}

func TestContextHandlerSignature(t *testing.T) {
	// Force a member route for DELETE via a custom path so /:id is appended.
	specs := splitActions("/widgets", mustActions(t, &ctxRoute{}))
	h := mount(specs)

	if w := do(h, "GET", "/widgets"); w.Body.String() != "ctx-ok" {
		t.Errorf("GET /widgets = %q, want %q", w.Body.String(), "ctx-ok")
	}
	if w := do(h, "DELETE", "/widgets/42"); w.Body.String() != "deleted:42" {
		t.Errorf("DELETE /widgets/42 = %q, want %q", w.Body.String(), "deleted:42")
	}
}

func TestMemberCollectionSplit(t *testing.T) {
	specs := splitActions("/users", mustActions(t, &ctxRoute{}))
	byMethod := map[string]string{}
	for _, s := range specs {
		byMethod[s.Method] = s.Pattern
	}
	if byMethod["GET"] != "/users" {
		t.Errorf("GET pattern = %q, want /users", byMethod["GET"])
	}
	if byMethod["DELETE"] != "/users/:id" {
		t.Errorf("DELETE pattern = %q, want /users/:id", byMethod["DELETE"])
	}
}

func TestDiscoverFactoryAndFunc(t *testing.T) {
	registry := core.NewRegistry()
	core.ProvideValue[string](registry, "injected")

	Register(func(r *core.Registry) any {
		_ = core.MustInvoke[string](r) // exercise DI resolution
		return &usersRoute{getBody: "g", postBody: "p"}
	})
	RegisterFunc("GET", func(c core.Context) error { return c.String(200, "fn") })

	specs, err := Discover(registry)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(specs) == 0 {
		t.Fatal("expected specs from Discover")
	}
	// Both the struct GET/POST and the function GET should resolve to "/" (this
	// test package's relative routes path is empty → "/").
	h := mount(specs)
	if w := do(h, "POST", "/"); w.Body.String() != "p" {
		t.Errorf("POST / = %q, want p", w.Body.String())
	}
}

func TestCustomPathValidation(t *testing.T) {
	_, err := basePatternFor(&badPathRoute{}, nil)
	if err == nil || !strings.Contains(err.Error(), "must return a path starting with '/'") {
		t.Fatalf("expected leading-slash validation error, got %v", err)
	}
}

type badPathRoute struct{}

func (*badPathRoute) Path() string             { return "no-leading-slash" }
func (*badPathRoute) GET(_ core.Context) error { return nil }

func TestParamExtraction(t *testing.T) {
	req := httptest.NewRequest("GET", "/users/42", nil)
	req = core.WithParams(req, map[string]any{"id": "42"})
	if core.Param(req, "id") != "42" {
		t.Errorf("Param id = %q, want 42", core.Param(req, "id"))
	}
	if core.Param(req, "missing") != "" {
		t.Errorf("Param missing = %q, want empty", core.Param(req, "missing"))
	}
}

func mustActions(t *testing.T, h any) []discoveredAction {
	t.Helper()
	a, err := discoverActions(h)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}
	return a
}

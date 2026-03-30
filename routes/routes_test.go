package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/patternmatch"
	"github.com/samber/do/v2"
)

// usersRoute simulates a route struct for /users.
type usersRoute struct {
	getBody  string
	postBody string
}

func (r *usersRoute) GET(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(r.getBody)) //nolint:errcheck
}
func (r *usersRoute) POST(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(r.postBody)) //nolint:errcheck
}

// userByIDRoute simulates a route struct for /users/:id.
type userByIDRoute struct{}

func (r *userByIDRoute) GET(w http.ResponseWriter, req *http.Request) {
	id := Param(req, "id")
	w.Write([]byte("user:" + id)) //nolint:errcheck
}
func (r *userByIDRoute) DELETE(w http.ResponseWriter, req *http.Request) {
	id := Param(req, "id")
	w.Write([]byte("deleted:" + id)) //nolint:errcheck
}

func TestRouter_StaticRoutes(t *testing.T) {
	router := New()

	route := &usersRoute{getBody: "list-users", postBody: "created"}
	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}
	for _, action := range discovered {
		addRoute(router, "/users", action.Method, action.Handler)
	}

	// GET /users
	{
		req := httptest.NewRequest("GET", "/users", nil)
		handler, params, ok := router.Match(req)
		if !ok {
			t.Fatal("expected match for GET /users")
		}
		w := httptest.NewRecorder()
		req = WithParams(req, params)
		handler(w, req)
		if w.Body.String() != "list-users" {
			t.Errorf("GET /users body = %q, want %q", w.Body.String(), "list-users")
		}
	}

	// POST /users
	{
		req := httptest.NewRequest("POST", "/users", nil)
		handler, params, ok := router.Match(req)
		if !ok {
			t.Fatal("expected match for POST /users")
		}
		w := httptest.NewRecorder()
		req = WithParams(req, params)
		handler(w, req)
		if w.Body.String() != "created" {
			t.Errorf("POST /users body = %q, want %q", w.Body.String(), "created")
		}
	}

	// PUT /users → 405
	{
		req := httptest.NewRequest("PUT", "/users", nil)
		handler, _, ok := router.Match(req)
		if !ok {
			t.Fatal("expected 405 match for PUT /users")
		}
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("PUT /users status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
		if w.Header().Get("Allow") == "" {
			t.Error("expected Allow header on 405")
		}
	}

	// GET /nonexistent → no match
	{
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		_, _, ok := router.Match(req)
		if ok {
			t.Error("expected no match for GET /nonexistent")
		}
	}
}

func TestRouter_DynamicRoutes(t *testing.T) {
	router := New()

	route := &userByIDRoute{}
	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}
	for _, action := range discovered {
		addRoute(router, "/users/:id", action.Method, action.Handler)
	}

	// GET /users/42
	{
		req := httptest.NewRequest("GET", "/users/42", nil)
		handler, params, ok := router.Match(req)
		if !ok {
			t.Fatal("expected match for GET /users/42")
		}
		w := httptest.NewRecorder()
		req = WithParams(req, params)
		handler(w, req)
		if w.Body.String() != "user:42" {
			t.Errorf("GET /users/42 body = %q, want %q", w.Body.String(), "user:42")
		}
	}

	// DELETE /users/42
	{
		req := httptest.NewRequest("DELETE", "/users/42", nil)
		handler, params, ok := router.Match(req)
		if !ok {
			t.Fatal("expected match for DELETE /users/42")
		}
		w := httptest.NewRecorder()
		req = WithParams(req, params)
		handler(w, req)
		if w.Body.String() != "deleted:42" {
			t.Errorf("DELETE /users/42 body = %q, want %q", w.Body.String(), "deleted:42")
		}
	}
}

func TestRouter_InitInterface(t *testing.T) {
	injector := do.New()
	do.Provide(injector, func(_ do.Injector) (string, error) {
		return "injected-value", nil
	})

	route := &initRoute{}
	if err := route.Init(injector); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if route.Value != "injected-value" {
		t.Errorf("Value = %q, want %q", route.Value, "injected-value")
	}
}

type initRoute struct {
	Value string
}

func (r *initRoute) Init(injector do.Injector) error {
	r.Value = do.MustInvoke[string](injector)
	return nil
}

func (r *initRoute) GET(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(r.Value)) //nolint:errcheck
}

func TestRouter_ParamExtraction(t *testing.T) {
	req := httptest.NewRequest("GET", "/users/42", nil)
	req = WithParams(req, map[string]any{"id": "42"})

	if Param(req, "id") != "42" {
		t.Errorf("Param(req, 'id') = %q, want %q", Param(req, "id"), "42")
	}
	if Param(req, "missing") != "" {
		t.Errorf("Param(req, 'missing') = %q, want empty", Param(req, "missing"))
	}
}

// ── Pather (custom path override) tests ──────────────────────────────────────

type customPathRoute struct{}

func (*customPathRoute) Path() string { return "/api/v2/hooks" }
func (r *customPathRoute) POST(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("hook-received")) //nolint:errcheck
}

func TestRouter_CustomPath(t *testing.T) {
	router := New()
	route := &customPathRoute{}

	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}

	for _, action := range discovered {
		addRoute(router, route.Path(), action.Method, action.Handler)
	}

	req := httptest.NewRequest("POST", "/api/v2/hooks", nil)
	handler, _, matched := router.Match(req)
	if !matched {
		t.Fatal("expected match for POST /api/v2/hooks")
	}
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Body.String() != "hook-received" {
		t.Errorf("body = %q, want %q", w.Body.String(), "hook-received")
	}
}

type dynamicCustomRoute struct{}

func (*dynamicCustomRoute) Path() string { return "/widgets/:widget_id/parts" }
func (r *dynamicCustomRoute) GET(w http.ResponseWriter, req *http.Request) {
	wid := Param(req, "widget_id")
	w.Write([]byte("parts-for:" + wid)) //nolint:errcheck
}

func TestRouter_CustomPath_Dynamic(t *testing.T) {
	router := New()
	route := &dynamicCustomRoute{}

	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}
	for _, action := range discovered {
		addRoute(router, route.Path(), action.Method, action.Handler)
	}

	req := httptest.NewRequest("GET", "/widgets/99/parts", nil)
	handler, params, matched := router.Match(req)
	if !matched {
		t.Fatal("expected match for GET /widgets/99/parts")
	}
	w := httptest.NewRecorder()
	req = WithParams(req, params)
	handler(w, req)
	if w.Body.String() != "parts-for:99" {
		t.Errorf("body = %q, want %q", w.Body.String(), "parts-for:99")
	}
}

type badPathRoute struct{}

func (*badPathRoute) Path() string                                   { return "no-leading-slash" }
func (r *badPathRoute) GET(w http.ResponseWriter, req *http.Request) {}

func TestRouter_CustomPath_ValidationError(t *testing.T) {
	// Build() should reject a Pather that returns a path without leading "/".
	pending = append(pending, &badPathRoute{})

	router := New()
	injector := do.New()
	err := router.Build(injector)
	if err == nil {
		t.Fatal("expected error for path without leading '/'")
	}
	if !strings.Contains(err.Error(), "must return a path starting with '/'") {
		t.Errorf("unexpected error: %v", err)
	}
}

// addRoute is a test helper that adds a route directly to the router.
func addRoute(r *Router, pattern, method string, handler http.HandlerFunc) {
	compiled := patternmatch.CompilePattern(pattern)
	ar := &apiRoute{
		pattern:  pattern,
		method:   method,
		compiled: compiled,
		handler:  handler,
	}
	if compiled.IsStatic {
		r.staticRoutes[pattern] = append(r.staticRoutes[pattern], ar)
	} else {
		r.dynamicRoutes = append(r.dynamicRoutes, ar)
		sortAPIRoutes(r.dynamicRoutes)
	}
}

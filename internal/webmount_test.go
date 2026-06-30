package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/routes"
	"github.com/polagonow/pola/webframework/std"
)

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeRenderer struct{}

func (fakeRenderer) Name() string                    { return "fake" }
func (fakeRenderer) FileExtensions() []string        { return nil }
func (fakeRenderer) Capabilities() []core.Capability { return nil }
func (fakeRenderer) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("PAGE"))
}

type fakeRouter struct{ pages map[string]bool }

func (fakeRouter) Name() string { return "fake" }
func (fakeRouter) ScanRoutes(context.Context, core.FS, string, []string) ([]core.Route, error) {
	return nil, nil
}
func (f fakeRouter) Resolve(_ context.Context, path string) (*core.Route, map[string]any) {
	if f.pages[path] {
		return &core.Route{Pattern: path}, map[string]any{}
	}
	return nil, nil
}

type fakeCache struct{ invalidated []string }

func (fakeCache) Name() string                                                 { return "fake" }
func (fakeCache) Get(context.Context, string) ([]byte, bool, error)            { return nil, false, nil }
func (fakeCache) Set(context.Context, string, []byte, core.CacheOptions) error { return nil }
func (fakeCache) Delete(context.Context, string) error                         { return nil }
func (c *fakeCache) Invalidate(_ context.Context, prefix string) error {
	c.invalidated = append(c.invalidated, prefix)
	return nil
}
func (fakeCache) Clear(context.Context) error { return nil }

// ── Tests ────────────────────────────────────────────────────────────────────

func mountTest(d mountDeps) http.Handler {
	d.fw = std.New()
	return mountApp(d)
}

func req(h http.Handler, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestMount_APIAndPagePrecedence(t *testing.T) {
	cache := &fakeCache{}
	specs := routes.RouteSpecs{
		{Method: "GET", Pattern: "/about", Handler: func(c core.Context) error { return c.String(200, "api-about") }},
		{Method: "GET", Pattern: "/items", Handler: func(c core.Context) error { return c.String(200, "api-items") }},
		{Method: "POST", Pattern: "/items", Handler: func(c core.Context) error { return c.String(201, "created") }},
		{Method: "DELETE", Pattern: "/items/:id", Handler: func(c core.Context) error { return c.String(200, "del:"+c.Param("id")) }},
	}
	h := mountTest(mountDeps{
		renderer: fakeRenderer{},
		router:   fakeRouter{pages: map[string]bool{"/about": true}},
		cache:    cache,
		specs:    specs,
	})

	// Page wins for GET /about even though an API GET is registered.
	if w := req(h, "GET", "/about"); w.Body.String() != "PAGE" {
		t.Errorf("GET /about = %q, want PAGE (page precedence)", w.Body.String())
	}
	// API GET wins when no page exists.
	if w := req(h, "GET", "/items"); w.Body.String() != "api-items" {
		t.Errorf("GET /items = %q, want api-items", w.Body.String())
	}
	// API POST runs and triggers cache invalidation on 2xx.
	if w := req(h, "POST", "/items"); w.Code != 201 || w.Body.String() != "created" {
		t.Errorf("POST /items = %d %q, want 201 created", w.Code, w.Body.String())
	}
	// Dynamic member route with param extraction.
	if w := req(h, "DELETE", "/items/42"); w.Body.String() != "del:42" {
		t.Errorf("DELETE /items/42 = %q, want del:42", w.Body.String())
	}
	// Unknown GET → fallback renderer.
	if w := req(h, "GET", "/nope"); w.Body.String() != "PAGE" {
		t.Errorf("GET /nope = %q, want PAGE (fallback)", w.Body.String())
	}
	// 405 when method not registered on an existing path.
	if w := req(h, "PUT", "/items"); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT /items = %d, want 405", w.Code)
	}
	// Cache invalidated after the successful POST (and DELETE).
	if len(cache.invalidated) == 0 {
		t.Error("expected SSR cache invalidation after mutations")
	}
}

func TestMount_ReservedMount(t *testing.T) {
	metrics := &fakeMetrics{path: "/_pola/metrics"}
	h := mountTest(mountDeps{
		renderer: fakeRenderer{},
		router:   fakeRouter{},
		metrics:  metrics,
	})
	if w := req(h, "GET", "/_pola/metrics"); w.Body.String() != "METRICS" {
		t.Errorf("GET /_pola/metrics = %q, want METRICS", w.Body.String())
	}
}

func TestMount_APIOnly404(t *testing.T) {
	h := mountTest(mountDeps{
		specs: routes.RouteSpecs{
			{Method: "GET", Pattern: "/ping", Handler: func(c core.Context) error { return c.String(200, "pong") }},
		},
	})
	if w := req(h, "GET", "/ping"); w.Body.String() != "pong" {
		t.Errorf("GET /ping = %q, want pong", w.Body.String())
	}
	if w := req(h, "GET", "/missing"); w.Code != http.StatusNotFound {
		t.Errorf("GET /missing = %d, want 404", w.Code)
	}
}

type fakeMetrics struct{ path string }

func (m *fakeMetrics) Name() string                                     { return "fake" }
func (m *fakeMetrics) Path() string                                     { return m.path }
func (m *fakeMetrics) RecordRequest(string, string, int, time.Duration) {}
func (m *fakeMetrics) RecordRender(string, time.Duration)               {}
func (m *fakeMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("METRICS")) })
}

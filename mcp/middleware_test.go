package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatchesMount(t *testing.T) {
	cases := []struct {
		path, mount string
		want        bool
	}{
		{"/mcp", "/mcp", true},
		{"/mcp/foo", "/mcp", true},
		{"/mcp/foo/bar", "/mcp", true},
		{"/mcpish", "/mcp", false},
		{"/other", "/mcp", false},
		{"/", "/mcp", false},
		{"", "/mcp", false},
	}
	for _, c := range cases {
		if got := matchesMount(c.path, c.mount); got != c.want {
			t.Errorf("matchesMount(%q, %q) = %v, want %v", c.path, c.mount, got, c.want)
		}
	}
}

func TestMiddlewareShortCircuitsAtMount(t *testing.T) {
	mcpCalled := false
	mcpHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mcpCalled = true
		w.WriteHeader(http.StatusOK)
	})

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
	})

	mw := &mcpMiddleware{mount: "/mcp", handler: mcpHandler}
	wrapped := mw.Wrap(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/mcp", nil)
	wrapped.ServeHTTP(rec, req)
	if !mcpCalled {
		t.Fatal("expected MCP handler to be called for /mcp")
	}
	if nextCalled {
		t.Fatal("next handler should not run when MCP handles the request")
	}

	mcpCalled, nextCalled = false, false
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/other", nil)
	wrapped.ServeHTTP(rec, req)
	if mcpCalled {
		t.Fatal("MCP handler should not run for non-mount paths")
	}
	if !nextCalled {
		t.Fatal("next handler should run for non-mount paths")
	}
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status: got %d, want 418", rec.Code)
	}
}

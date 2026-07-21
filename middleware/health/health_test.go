package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestLiveness(t *testing.T) {
	h := New().Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("liveness must short-circuit, not fall through")
	}))
	rec := do(h, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Errorf("liveness = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}
}

func TestReadinessPasses(t *testing.T) {
	h := New(WithCheck("db", func(context.Context) error { return nil })).
		Wrap(http.NotFoundHandler())
	rec := do(h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK {
		t.Errorf("readiness = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"db":"ok"`) {
		t.Errorf("body = %q, want db:ok", rec.Body.String())
	}
}

func TestReadinessFails(t *testing.T) {
	h := New(
		WithCheck("db", func(context.Context) error { return nil }),
		WithCheck("cache", func(context.Context) error { return errors.New("connection refused") }),
	).Wrap(http.NotFoundHandler())

	rec := do(h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("readiness with failing check = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "connection refused") {
		t.Errorf("body = %q, want the failing check's error", rec.Body.String())
	}
}

func TestPassThrough(t *testing.T) {
	var reached bool
	h := New().Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := do(h, http.MethodGet, "/api/users")
	if !reached || rec.Code != http.StatusTeapot {
		t.Errorf("non-health path should pass through (reached=%v code=%d)", reached, rec.Code)
	}
}

func TestCustomPaths(t *testing.T) {
	h := New(WithLivePath("/livez"), WithReadyPath("/ready")).Wrap(http.NotFoundHandler())
	if rec := do(h, http.MethodGet, "/livez"); rec.Code != http.StatusOK {
		t.Errorf("custom live path = %d, want 200", rec.Code)
	}
	// The default path must no longer be intercepted.
	if rec := do(h, http.MethodGet, "/healthz"); rec.Code != http.StatusNotFound {
		t.Errorf("default path should pass through when overridden, got %d", rec.Code)
	}
}

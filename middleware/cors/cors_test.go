package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okNext records whether the wrapped handler was reached and returns 200.
func okNext(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestPassthroughWhenNoOrigin(t *testing.T) {
	var reached bool
	h := New().Wrap(okNext(&reached))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !reached {
		t.Error("handler should run for a same-origin (no Origin header) request")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no CORS header expected, got %q", got)
	}
}

func TestPreflightWildcard(t *testing.T) {
	var reached bool
	h := New().Wrap(okNext(&reached))

	req := httptest.NewRequest(http.MethodOptions, "/api/x", nil)
	req.Header.Set("Origin", "https://foo.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type, x-custom")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached {
		t.Error("preflight must not fall through to the handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin = %q, want *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Allow-Methods should be set on preflight")
	}
	// "*" headers policy reflects the requested headers.
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "content-type, x-custom" {
		t.Errorf("Allow-Headers = %q, want reflected request headers", got)
	}
}

func TestActualRequestAllowedOrigin(t *testing.T) {
	var reached bool
	h := New(WithAllowedOrigins("https://ok.example")).Wrap(okNext(&reached))

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Origin", "https://ok.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached {
		t.Error("handler should run for an allowed cross-origin request")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ok.example" {
		t.Errorf("Allow-Origin = %q, want the concrete origin", got)
	}
	if got := rec.Header().Get("Vary"); got == "" {
		t.Error("Vary: Origin should be set when echoing a concrete origin")
	}
}

func TestDisallowedOrigin(t *testing.T) {
	var reached bool
	h := New(WithAllowedOrigins("https://ok.example")).Wrap(okNext(&reached))

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached {
		t.Error("actual request should still run; the browser enforces CORS")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no Allow-Origin expected for a disallowed origin, got %q", got)
	}
}

func TestCredentialsReflectsOrigin(t *testing.T) {
	var reached bool
	h := New(WithAllowCredentials()).Wrap(okNext(&reached)) // wildcard origins + credentials

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Origin", "https://foo.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://foo.example" {
		t.Errorf("Allow-Origin = %q, want concrete origin (never * with credentials)", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials = %q, want true", got)
	}
}

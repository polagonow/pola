package securityheaders_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/middleware/securityheaders"
)

func TestSecurityHeaders_SetsAllHeaders(t *testing.T) {
	mw := securityheaders.New()
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	exact := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}
	for header, want := range exact {
		if got := rr.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	// CSP includes a random nonce, so check prefix and nonce presence.
	csp := rr.Header().Get("Content-Security-Policy")
	if !strings.HasPrefix(csp, "default-src 'self'; script-src 'self' 'nonce-") {
		t.Errorf("CSP = %q, want prefix with nonce", csp)
	}
}

func TestSecurityHeaders_DevMode_RelaxesCSP(t *testing.T) {
	mw := securityheaders.New(securityheaders.WithDev())
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'unsafe-inline'") {
		t.Errorf("dev CSP should contain 'unsafe-inline', got %q", csp)
	}
	if !strings.Contains(csp, "'nonce-") {
		t.Errorf("dev CSP should contain nonce, got %q", csp)
	}
}

func TestSecurityHeaders_SetsNonceInContext(t *testing.T) {
	mw := securityheaders.New()
	var nonce string
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce, _ = r.Context().Value(core.NonceContextKey).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if nonce == "" {
		t.Fatal("expected nonce in context")
	}
	csp := rr.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'nonce-"+nonce+"'") {
		t.Errorf("CSP nonce mismatch: nonce=%q, CSP=%q", nonce, csp)
	}
}

func TestSecurityHeaders_Name(t *testing.T) {
	mw := securityheaders.New()
	if mw.Name() != "securityheaders" {
		t.Fatalf("expected 'securityheaders', got %q", mw.Name())
	}
}

func TestSecurityHeaders_PassesThrough(t *testing.T) {
	mw := securityheaders.New()
	called := false
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

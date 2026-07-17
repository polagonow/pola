package requireauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authjwt "github.com/polagonow/pola/auth/jwt"
	"github.com/polagonow/pola/middleware/requireauth"
)

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestUnauthenticated_ProtectedRedirects(t *testing.T) {
	mw := requireauth.New(
		requireauth.WithSecret([]byte("s")),
		requireauth.WithProtect("/dashboard"),
		requireauth.WithRedirect("/sign-in"),
	)
	h := mw.Wrap(handler())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/dashboard/settings", nil))
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/sign-in" {
		t.Fatalf("expected 303 -> /sign-in, got %d -> %q", rr.Code, rr.Header().Get("Location"))
	}
}

func TestUnprotectedPath_PassesThrough(t *testing.T) {
	mw := requireauth.New(requireauth.WithSecret([]byte("s")), requireauth.WithProtect("/dashboard"))
	h := mw.Wrap(handler())
	for _, p := range []string{"/", "/pricing", "/dashboardxyz"} { // note: not a subpath of /dashboard
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("path %q should pass through, got %d", p, rr.Code)
		}
	}
}

func TestAuthenticated_ProtectedPasses(t *testing.T) {
	secret := []byte("s3cr3t")
	mw := requireauth.New(
		requireauth.WithSecret(secret),
		requireauth.WithProtect("/dashboard"),
		requireauth.WithRedirect("/sign-in"),
	)
	h := mw.Wrap(handler())

	tok, _ := authjwt.Sign(map[string]any{"userId": 1}, secret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: tok})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("authenticated request should pass, got %d", rr.Code)
	}
}

func TestName(t *testing.T) {
	if requireauth.New().Name() != "require-auth" {
		t.Fatal("expected name require-auth")
	}
}

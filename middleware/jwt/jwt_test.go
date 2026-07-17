package jwt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mwjwt "github.com/polagonow/pola/middleware/jwt"
)

func secretMW() interface {
	Name() string
	Wrap(http.Handler) http.Handler
} {
	return mwjwt.New(mwjwt.WithSecret([]byte("test-secret")), mwjwt.WithSecure(false))
}

func TestName(t *testing.T) {
	if secretMW().Name() != "jwt" {
		t.Fatal("expected middleware name 'jwt'")
	}
}

func TestSet_IssuesCookie_And_GetSeesIt(t *testing.T) {
	mw := secretMW()
	var readBack map[string]any
	h := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mwjwt.Set(r.Context(), map[string]any{"user": map[string]any{"id": float64(7)}})
		// Within the same request, Get should reflect the just-set claims.
		readBack = mwjwt.Get(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if u, _ := readBack["user"].(map[string]any); u == nil || u["id"] != float64(7) {
		t.Fatalf("Get did not reflect Set: %#v", readBack)
	}
	var sessionCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected a non-empty 'session' cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}

	// A follow-up request carrying the cookie should decode the same claims.
	var got map[string]any
	h2 := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = mwjwt.Get(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)
	h2.ServeHTTP(httptest.NewRecorder(), req)

	if u, _ := got["user"].(map[string]any); u == nil || u["id"] != float64(7) {
		t.Fatalf("follow-up Get did not decode claims: %#v", got)
	}
}

func TestClear_DeletesCookie(t *testing.T) {
	mw := secretMW()
	h := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mwjwt.Clear(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	for _, c := range rr.Result().Cookies() {
		if c.Name == "session" && c.MaxAge >= 0 {
			t.Fatalf("expected session cookie to be expired (MaxAge<0), got %d", c.MaxAge)
		}
	}
}

func TestGet_NoSession_ReturnsEmpty(t *testing.T) {
	if len(mwjwt.Get(context.Background())) != 0 {
		t.Fatal("expected empty claims with no middleware context")
	}
}

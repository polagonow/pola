package csrf_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/middleware/csrf"
)

func TestCSRF_Name(t *testing.T) {
	mw := csrf.New(csrf.WithSecure(false))
	if mw.Name() != "csrf" {
		t.Fatalf("expected 'csrf', got %q", mw.Name())
	}
}

func TestCSRF_GET_SetsTokenHeader(t *testing.T) {
	mw := csrf.New(csrf.WithSecure(false))
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	token := rr.Header().Get("X-CSRF-Token")
	if token == "" {
		t.Fatal("expected X-CSRF-Token header to be set")
	}
}

func TestCSRF_POST_WithoutToken_Returns403(t *testing.T) {
	mw := csrf.New(csrf.WithSecure(false))
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("data=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestCSRF_POST_WithToken_Succeeds(t *testing.T) {
	mw := csrf.New(csrf.WithSecure(false))
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Step 1: GET to obtain the token and cookie.
	getReq := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	getRR := httptest.NewRecorder()
	handler.ServeHTTP(getRR, getReq)

	token := getRR.Header().Get("X-CSRF-Token")
	if token == "" {
		t.Fatal("expected X-CSRF-Token header from GET")
	}

	// Extract Set-Cookie from GET response.
	cookies := getRR.Result().Cookies()

	// Step 2: POST with the token header and cookie.
	postReq := httptest.NewRequest(http.MethodPost, "http://localhost/submit", strings.NewReader("data=test"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-CSRF-Token", token)
	postReq.Header.Set("Origin", "https://localhost")
	for _, c := range cookies {
		postReq.AddCookie(c)
	}
	postRR := httptest.NewRecorder()
	handler.ServeHTTP(postRR, postReq)

	if postRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", postRR.Code, postRR.Body.String())
	}
}

func TestCSRF_CustomCookieName(t *testing.T) {
	mw := csrf.New(csrf.WithSecure(false), csrf.WithCookieName("my_csrf"))
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "my_csrf" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected cookie named 'my_csrf'")
	}
}

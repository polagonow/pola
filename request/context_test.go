package request_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polagonow/pola/request"
)

func TestRequestInfo_Accessors(t *testing.T) {
	mw := request.Plugin()
	// The plugin registers the middleware into a registry; exercise the wrapped
	// handler directly by re-registering through a tiny fake.
	var got struct {
		ip, ua, host, hdr string
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		got.ip = request.ClientIP(ctx)
		got.ua = request.UserAgent(ctx)
		got.host = request.Host(ctx)
		got.hdr = request.Header(ctx, "X-Custom")
		w.WriteHeader(http.StatusOK)
	})

	_ = mw
	wrapped := request.Middleware().Wrap(handler)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/x", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	req.Header.Set("User-Agent", "pola-test")
	req.Header.Set("X-Custom", "hello")
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	if got.ip != "203.0.113.7" {
		t.Fatalf("ClientIP = %q, want first XFF hop", got.ip)
	}
	if got.ua != "pola-test" {
		t.Fatalf("UserAgent = %q", got.ua)
	}
	if got.host != "example.test" {
		t.Fatalf("Host = %q", got.host)
	}
	if got.hdr != "hello" {
		t.Fatalf("Header = %q", got.hdr)
	}
}

func TestRequestInfo_NoMiddleware_Empty(t *testing.T) {
	if request.ClientIP(context.Background()) != "" {
		t.Fatal("expected empty ClientIP without middleware")
	}
	if _, ok := request.FromContext(context.Background()); ok {
		t.Fatal("expected no Info without middleware")
	}
}

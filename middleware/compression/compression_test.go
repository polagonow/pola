package compression_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polagonow/pola/middleware/compression"
)

func TestCompression_AppliesGzip(t *testing.T) {
	mw := compression.New()

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", rr.Header().Get("Content-Encoding"))
	}

	gr, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(body))
	}
}

func TestCompression_NoGzip_WhenNotAccepted(t *testing.T) {
	mw := compression.New()

	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain text"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding header.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("should not apply gzip when not accepted")
	}
	if rr.Body.String() != "plain text" {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestCompression_Name(t *testing.T) {
	mw := compression.New()
	if mw.Name() != "compression" {
		t.Fatalf("expected 'compression', got %q", mw.Name())
	}
}

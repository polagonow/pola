package filesystem_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/polagonow/pola/static/filesystem"
)

func TestHandler_ServesFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("static content")
	// Use a non-index filename to avoid http.FileServer's clean-URL redirect.
	if err := os.WriteFile(filepath.Join(dir, "style.css"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	srv := filesystem.New(dir)
	handler := srv.Handler("/static/")

	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != string(content) {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestHandler_404ForMissing(t *testing.T) {
	srv := filesystem.New(t.TempDir())
	handler := srv.Handler("/static/")

	req := httptest.NewRequest(http.MethodGet, "/static/missing.txt", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

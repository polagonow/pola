package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTMLShell_HasRootDiv(t *testing.T) {
	body := page(t, "/")
	if !strings.Contains(body, `id="root"`) {
		t.Errorf("HTML shell missing <div id=\"root\">")
	}
}

func TestHTMLShell_HasClientEntryScript(t *testing.T) {
	body := page(t, "/")
	if !strings.Contains(body, `type="module"`) {
		t.Errorf("HTML shell missing <script type=\"module\">")
	}
}

func TestHTMLShell_ContentType(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	ct := w.Result().Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %q", ct)
	}
}

func TestHTMLShell_404(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Result().StatusCode)
	}
}

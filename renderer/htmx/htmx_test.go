package htmx_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/htmx"
)

func TestRendererName(t *testing.T) {
	r := htmx.New()
	if got := r.Name(); got != "htmx" {
		t.Errorf("Name() = %q, want %q", got, "htmx")
	}
}

func TestRendererFileExtensions(t *testing.T) {
	r := htmx.New()
	exts := r.FileExtensions()
	want := map[string]bool{".html": true, ".templ": true}
	if len(exts) != len(want) {
		t.Fatalf("FileExtensions() returned %d extensions, want %d", len(exts), len(want))
	}
	for _, ext := range exts {
		if !want[ext] {
			t.Errorf("unexpected extension %q in FileExtensions()", ext)
		}
	}
}

func TestRendererImplementsInterface(t *testing.T) {
	var _ core.Renderer = htmx.New()
	var _ core.RenderDepsAware = htmx.New()
}

func TestRendererCapabilities(t *testing.T) {
	r := htmx.New()
	caps := r.Capabilities()
	found := map[string]bool{}
	for _, c := range caps {
		found[string(c)] = true
	}
	if !found["server-side"] {
		t.Error("missing server-side capability")
	}
	if !found["htmx"] {
		t.Error("missing htmx capability")
	}
}

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func serveWithRoute(t *testing.T, r *htmx.Renderer, route *core.Route, props map[string]any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	ctx := req.Context()
	ctx = core.WithRenderRequest(ctx, route, props, nil, http.StatusOK, nil)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRenderHTMLTemplate(t *testing.T) {
	dir := t.TempDir()
	tmplPath := writeTemplate(t, dir, "page.html", `<div id="content"><h1>{{.title}}</h1></div>`)

	r := htmx.New()
	route := &core.Route{Pattern: "/test", Export: tmplPath}
	props := map[string]any{"title": "Hello World"}

	rec := serveWithRoute(t, r, route, props, nil)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<h1>Hello World</h1>") {
		t.Errorf("body missing rendered content: %s", body)
	}
	if !strings.Contains(body, "<!DOCTYPE html>") || !strings.Contains(body, "</html>") {
		t.Errorf("full-page response should include HTML shell: %s", body)
	}
}

func TestHTMXPartialRequest(t *testing.T) {
	dir := t.TempDir()
	tmplPath := writeTemplate(t, dir, "partial.html", `<div id="result">{{.name}}</div>`)

	r := htmx.New()
	route := &core.Route{Pattern: "/api/search", Export: tmplPath}
	props := map[string]any{"name": "Pola"}

	rec := serveWithRoute(t, r, route, props, map[string]string{"HX-Request": "true"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<div id="result">Pola</div>`) {
		t.Errorf("partial response wrong: %s", body)
	}
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTMX partial should NOT include the full HTML shell")
	}
	if rec.Header().Get("Vary") != "HX-Request" {
		t.Errorf("Vary header = %q, want HX-Request", rec.Header().Get("Vary"))
	}
}

func TestNotFoundRoute(t *testing.T) {
	r := htmx.New()

	req := httptest.NewRequest("GET", "/missing", nil)
	ctx := core.WithRenderRequest(req.Context(), nil, nil, nil, 0, nil)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Not Found") {
		t.Errorf("404 body should contain 'Not Found': %s", body)
	}
}

func TestInvalidateTemplates(t *testing.T) {
	dir := t.TempDir()
	tmplPath := writeTemplate(t, dir, "cached.html", `<p>version1</p>`)

	r := htmx.New()
	route := &core.Route{Pattern: "/cached", Export: tmplPath}

	rec := serveWithRoute(t, r, route, nil, map[string]string{"HX-Request": "true"})
	if !strings.Contains(rec.Body.String(), "version1") {
		t.Fatal("first render should show version1")
	}

	writeTemplate(t, dir, "cached.html", `<p>version2</p>`)
	r.InvalidateTemplates()

	rec = serveWithRoute(t, r, route, nil, map[string]string{"HX-Request": "true"})
	if !strings.Contains(rec.Body.String(), "version2") {
		t.Errorf("after invalidation should show version2: %s", rec.Body.String())
	}
}

func TestPlugin(t *testing.T) {
	p := htmx.Plugin()
	if got := p.Name(); got != "htmx" {
		t.Errorf("Plugin name = %q, want %q", got, "htmx")
	}
}

func TestTemplateFuncMap(t *testing.T) {
	dir := t.TempDir()
	tmplPath := writeTemplate(t, dir, "funcs.html", `<div>{{safeHTML "<strong>bold</strong>"}}</div>`)

	r := htmx.New()
	route := &core.Route{Pattern: "/funcs", Export: tmplPath}

	rec := serveWithRoute(t, r, route, nil, map[string]string{"HX-Request": "true"})
	body := rec.Body.String()
	if !strings.Contains(body, `<strong>bold</strong>`) {
		t.Errorf("safeHTML func not working: %s", body)
	}
}

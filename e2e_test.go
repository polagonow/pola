package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gojsx/build"
	"gojsx/runtime"
)

// testApp builds the full app once and returns a configured *App for reuse.
// Building is expensive (esbuild + Goja VM init) so we share it across tests.
var (
	_testApp *App
	_testErr error
)

func init() {
	_testApp, _testErr = newTestApp()
}

func newTestApp() (*App, error) {
	appDir := "./ui"

	pages, err := build.DiscoverPages(appDir)
	if err != nil {
		return nil, fmt.Errorf("discover pages: %w", err)
	}
	gc, err := build.DiscoverGlobalComponents(appDir)
	if err != nil {
		return nil, fmt.Errorf("discover global components: %w", err)
	}
	clientComponents, err := build.DiscoverClientComponents(appDir)
	if err != nil {
		return nil, fmt.Errorf("discover client components: %w", err)
	}
	seen := make(map[string]bool)
	for _, p := range pages {
		for _, seg := range p.Segments {
			if seg.ErrorPath != "" && !seen[seg.ErrorPath] {
				seen[seg.ErrorPath] = true
				clientComponents = append(clientComponents, seg.ErrorPath)
			}
		}
	}
	if gc.ErrorPath != "" && !seen[gc.ErrorPath] {
		seen[gc.ErrorPath] = true
		clientComponents = append(clientComponents, gc.ErrorPath)
	}

	bundleResult, err := build.Bundle(build.BundlerConfig{
		AppDir:             appDir,
		OutDir:             "./public/assets",
		ClientEntry:        "./ui/_client.tsx",
		Pages:              pages,
		ClientComponents:   clientComponents,
		GlobalNotFoundPath: gc.NotFoundPath,
		GlobalErrorPath:    gc.ErrorPath,
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
	}

	testCatalog := []map[string]any{
		{"id": 1, "name": "Widget Alpha", "price": 29.99, "stock": 142},
		{"id": 2, "name": "Widget Beta", "price": 49.99, "stock": 37},
		{"id": 3, "name": "Widget Gamma", "price": 9.99, "stock": 891},
		{"id": 4, "name": "Turbo Sprocket", "price": 199.00, "stock": 12},
	}
	bridge := runtime.BridgeConfig{
		Context: map[string]runtime.GoFunc{
			"getProducts": func(_ []any) (any, error) { return testCatalog, nil },
			"getProduct": func(args []any) (any, error) {
				id := ""
				if len(args) > 0 {
					id = fmt.Sprintf("%v", args[0])
				}
				for _, p := range testCatalog {
					if fmt.Sprintf("%v", p["id"]) == id {
						return p, nil
					}
				}
				return nil, fmt.Errorf("product %q not found", id)
			},
			"getUser": func(_ []any) (any, error) {
				return map[string]any{
					"id": "42", "name": "Jane Doe",
					"email": "jane@example.com", "role": "admin",
				}, nil
			},
			"query": func(_ []any) (any, error) { return []any{}, nil },
		},
	}

	serverJS, err := os.ReadFile(bundleResult.ServerBundlePath)
	if err != nil {
		return nil, fmt.Errorf("read server bundle: %w", err)
	}
	pool, err := runtime.NewVMPool(string(serverJS), bridge)
	if err != nil {
		return nil, fmt.Errorf("vm pool: %w", err)
	}

	manifest, err := runtime.LoadManifest(bundleResult.Manifest)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	globalNotFoundExport := ""
	if gc.NotFoundPath != "" {
		globalNotFoundExport = "GlobalNotFound"
	}
	app := &App{
		pool:                 pool,
		renderer:             runtime.NewRenderer(pool, manifest),
		clientEntryScript:    bundleResult.ClientEntryOutput,
		importURLs:           bundleResult.ImportURLs,
		globalNotFoundExport: globalNotFoundExport,
	}

	// Auto-register routes from discovered pages — mirrors main()
	for _, p := range pages {
		app.routes = append(app.routes, Route{
			Pattern: build.RoutePattern(appDir, p.PageComponentPath),
			Export:  build.PageAlias(p),
		})
	}

	return app, nil
}

func requireApp(t *testing.T) *App {
	t.Helper()
	if _testErr != nil {
		t.Fatalf("app init failed: %v", _testErr)
	}
	return _testApp
}

// rsc performs a GET <path> with Content-Type: text/x-component and returns the body.
func rsc(t *testing.T, path string) string {
	t.Helper()
	app := requireApp(t)
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("RSC GET %s → %d", path, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// page performs a plain GET <path> for the HTML shell.
func page(t *testing.T, path string) string {
	t.Helper()
	app := requireApp(t)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s → %d", path, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// ─── Flight wire format helpers ──────────────────────────────────────────────

// flightTree finds the first line starting with "0:" and unmarshals it as JSON.
func flightTree(t *testing.T, body string) any {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "0:") {
			var v any
			if err := json.Unmarshal([]byte(line[2:]), &v); err != nil {
				t.Fatalf("unmarshal flight line: %v\nbody: %s", err, line)
			}
			return v
		}
	}
	t.Fatalf("no 0: line in Flight output:\n%s", body)
	return nil
}

// flightContains reports whether the raw Flight body contains substr.
func flightContains(body, substr string) bool {
	return strings.Contains(body, substr)
}

// ─── HTML shell tests ─────────────────────────────────────────────────────────

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
	if !strings.Contains(body, `client`) {
		t.Errorf("HTML shell missing client script reference")
	}
}

func TestHTMLShell_HasNavLinks(t *testing.T) {
	body := page(t, "/")
	for _, href := range []string{`href="/"`, `href="/products"`, `href="/user`} {
		if !strings.Contains(body, href) {
			t.Errorf("HTML shell missing nav link %s", href)
		}
	}
}

func TestHTMLShell_ContentType(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	ct := w.Result().Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %q", ct)
	}
}

func TestHTMLShell_404(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Result().StatusCode)
	}
}

// ─── RSC Flight output tests ──────────────────────────────────────────────────

func TestRSC_ContentType(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	ct := w.Result().Header.Get("Content-Type")
	if !strings.Contains(ct, "text/x-component") {
		t.Errorf("expected text/x-component, got %q", ct)
	}
}

func TestRSC_IndexPage_FlightFormat(t *testing.T) {
	body := rsc(t, "/")
	if !strings.Contains(body, "0:") {
		t.Errorf("no Flight row in output:\n%s", body)
	}
	flightTree(t, body)
}

func TestRSC_IndexPage_HasPageContent(t *testing.T) {
	body := rsc(t, "/")
	for _, want := range []string{"GoJSX", "className", "page"} {
		if !flightContains(body, want) {
			t.Errorf("Flight output missing %q", want)
		}
	}
}

func TestRSC_IndexPage_HasProducts(t *testing.T) {
	body := rsc(t, "/")
	for _, product := range []string{"Widget Alpha", "Widget Beta", "Widget Gamma", "Turbo Sprocket"} {
		if !flightContains(body, product) {
			t.Errorf("IndexPage Flight missing product %q", product)
		}
	}
}

func TestRSC_ProductsPage(t *testing.T) {
	body := rsc(t, "/products")
	// Products are delivered in resolved Suspense chunks after the 0: row,
	// so search the full Flight body rather than only the parsed tree.
	for _, want := range []string{"Widget Alpha", "Widget Beta", "product-list"} {
		if !flightContains(body, want) {
			t.Errorf("ProductsPage missing %q in Flight body", want)
		}
	}
}

func TestRSC_ProductsPage_AllProducts(t *testing.T) {
	body := rsc(t, "/products")
	products := []string{"Widget Alpha", "Widget Beta", "Widget Gamma", "Turbo Sprocket"}
	for _, p := range products {
		if !flightContains(body, p) {
			t.Errorf("ProductsPage missing product %q", p)
		}
	}
}

func TestRSC_UserPage(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/user?id=42", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("RSC GET /user?id=42 → %d", w.Result().StatusCode)
	}
	body, _ := io.ReadAll(w.Result().Body)
	for _, want := range []string{"Jane Doe", "jane@example.com", "admin"} {
		if !flightContains(string(body), want) {
			t.Errorf("UserPage Flight missing %q", want)
		}
	}
}

func TestRSC_404(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.handleRoute(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown route, got %d", w.Result().StatusCode)
	}
}

func TestRSC_AboutPage(t *testing.T) {
	body := rsc(t, "/about")
	for _, want := range []string{"About", "synchronous", "Goja"} {
		if !flightContains(body, want) {
			t.Errorf("AboutPage Flight missing %q", want)
		}
	}
}

// ─── Flight tree structure tests ──────────────────────────────────────────────

// walkFlight recursively searches a decoded Flight tree for a string value.
func walkFlight(v any, target string) bool {
	switch val := v.(type) {
	case string:
		return strings.Contains(val, target)
	case []any:
		for _, item := range val {
			if walkFlight(item, target) {
				return true
			}
		}
	case map[string]any:
		for k, item := range val {
			if strings.Contains(k, target) || walkFlight(item, target) {
				return true
			}
		}
	}
	return false
}

func TestFlightTree_IndexPage_Structure(t *testing.T) {
	body := rsc(t, "/")
	tree := flightTree(t, body)
	arr, ok := tree.([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("expected array root, got %T: %v", tree, tree)
	}
	if arr[0] != "$" {
		t.Errorf("expected '$' as first element, got %v", arr[0])
	}
	// arr[1] is the element type — may be "div" (no layout) or a layout/error/suspense
	// wrapper reference when companion files are present. Only the "$" marker is stable.
}

func TestFlightTree_IndexPage_HasHeading(t *testing.T) {
	body := rsc(t, "/")
	tree := flightTree(t, body)
	if !walkFlight(tree, "GoJSX") {
		t.Error("IndexPage tree missing 'GoJSX' heading text")
	}
}

func TestFlightTree_ProductsPage_Structure(t *testing.T) {
	body := rsc(t, "/products")
	tree := flightTree(t, body)
	arr, ok := tree.([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("expected array root, got %T: %v", tree, tree)
	}
	// ProductsPage root is <Suspense> ("$1"), not a plain div.
	if arr[0] != "$" {
		t.Errorf("expected '$' as first element, got %v", arr[0])
	}
}

func TestFlightTree_UserPage_Structure(t *testing.T) {
	body := rsc(t, "/user")
	// Content may be in a deferred Suspense chunk; use full-body search.
	// Structural assertions only on the 0: row.
	tree := flightTree(t, body)
	arr, ok := tree.([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("expected array root, got %T: %v", tree, tree)
	}
	if arr[0] != "$" {
		t.Errorf("expected '$' as first element, got %v", arr[0])
	}
	// Content assertions use full body (data may be in a resolved Suspense chunk).
	if !flightContains(body, "Jane Doe") {
		t.Error("UserPage Flight body missing 'Jane Doe'")
	}
	if !flightContains(body, "admin") {
		t.Error("UserPage Flight body missing role 'admin'")
	}
}

// ─── Client bundle tests ──────────────────────────────────────────────────────

func TestClientBundle_Served(t *testing.T) {
	app := requireApp(t)
	if app.clientEntryScript == "" {
		t.Fatal("clientEntryScript is empty — esbuild client pass failed")
	}
	if !strings.HasPrefix(app.clientEntryScript, "/public/") {
		t.Errorf("clientEntryScript should start with /public/, got %q", app.clientEntryScript)
	}
}

func TestClientBundle_FileExists(t *testing.T) {
	app := requireApp(t)
	rel := strings.TrimPrefix(app.clientEntryScript, "/public/")
	path := "public/" + rel
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("client bundle file not found at %q: %v", path, err)
	}
	if info.Size() == 0 {
		t.Errorf("client bundle file is empty: %s", path)
	}
	if info.Size() < 10_000 {
		t.Errorf("client bundle suspiciously small (%d bytes) — React probably missing", info.Size())
	}
}

func TestClientBundle_NoWebpackRequire(t *testing.T) {
	app := requireApp(t)
	rel := strings.TrimPrefix(app.clientEntryScript, "/public/")
	path := "public/" + rel
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read client bundle: %v", err)
	}
	if strings.Contains(string(data), "__webpack_require__") {
		t.Error("client bundle contains __webpack_require__ — expected native ESM import()")
	}
}

// ─── Concurrent requests test ─────────────────────────────────────────────────

func TestRSC_Concurrent(t *testing.T) {
	app := requireApp(t)
	const n = 10
	results := make(chan error, n)

	for range n {
		go func() {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.handleRoute(w, req)
			body, _ := io.ReadAll(w.Result().Body)
			s := string(body)
			if !strings.Contains(s, "0:") {
				results <- fmt.Errorf("concurrent request got bad output: %s", s[:min(len(s), 100)])
			} else {
				results <- nil
			}
		}()
	}

	for range n {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}

func TestRSC_ProductDetailPage(t *testing.T) {
	app := requireApp(t)
	body := rsc(t, "/products/1")
	for _, want := range []string{"Widget Alpha", "29.99", "142"} {
		if !flightContains(body, want) {
			t.Errorf("product detail missing %q in Flight body", want)
		}
	}
	// Back link present
	if !flightContains(body, "/products") {
		t.Errorf("product detail missing back link")
	}
	_ = app
}

func TestRSC_ProductDetailPage_NotFound(t *testing.T) {
	app := requireApp(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/products/999", nil)
	req.Header.Set("Content-Type", "text/x-component")
	app.handleRoute(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	// React encodes thrown errors as a Flight error row: <id>:E{"digest":"..."}
	// With Suspense/Error wrappers the error may appear in any chunk, not only 0:.
	if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
		t.Errorf("expected Flight error row for unknown product, got: %s", string(body)[:min(len(string(body)), 200)])
	}
}

func TestRSC_NestedLayouts_AllPagesRender(t *testing.T) {
	// Verifies that nested layout wrapping (root → segment → page) does not break
	// any page and that each page returns a valid Flight 0: row.
	paths := []string{"/", "/about", "/products", "/products/1", "/user"}
	for _, p := range paths {
		body := rsc(t, p)
		if !strings.Contains(string(body), "0:") {
			t.Errorf("page %s: missing Flight 0: row", p)
		}
	}
}

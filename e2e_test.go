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
	bundleResult, err := build.Bundle(build.BundlerConfig{
		AppDir:      "./app",
		OutDir:      "./public/_pola_",
		ClientEntry: "./app/_client.tsx",
		PolyfillsJS: "./runtime/polyfills.js",
		Pages: []build.PageEntry{
			{File: "./app/pages/index.tsx", Export: "IndexPage"},
			{File: "./app/pages/products.tsx", Export: "ProductsPage"},
			{File: "./app/pages/user.tsx", Export: "UserPage"},
			{File: "./app/pages/about.tsx", Export: "AboutPage"},
		},
		ClientComponents: []string{
			"./app/components/Counter.tsx",
			"./app/components/ThemeToggle.tsx",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
	}

	bridge := runtime.BridgeConfig{
		Context: map[string]runtime.GoFunc{
			"getProducts": func(_ []interface{}) (any, error) {
				return []map[string]any{
					{"id": 1, "name": "Widget Alpha", "price": 29.99, "stock": 142},
					{"id": 2, "name": "Widget Beta", "price": 49.99, "stock": 37},
					{"id": 3, "name": "Widget Gamma", "price": 9.99, "stock": 891},
					{"id": 4, "name": "Turbo Sprocket", "price": 199.00, "stock": 12},
				}, nil
			},
			"getUser": func(_ []interface{}) (any, error) {
				return map[string]any{
					"id": "42", "name": "Jane Doe",
					"email": "jane@example.com", "role": "admin",
				}, nil
			},
			"query": func(_ []interface{}) (any, error) { return []any{}, nil },
		},
	}

	pool, err := runtime.NewVMPool(bundleResult.ServerBundle, bridge)
	if err != nil {
		return nil, fmt.Errorf("vm pool: %w", err)
	}

	manifest, err := runtime.LoadManifest(bundleResult.Manifest)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	app := &App{
		pool:              pool,
		renderer:          runtime.NewRenderer(pool, manifest),
		clientEntryScript: bundleResult.ClientEntryOutput,
	}

	// Register routes — mirrors main()
	app.Register(Route{
		Pattern: "/",
		Export:  "IndexPage",
		PropsFunc: func(r *http.Request) map[string]any {
			return map[string]any{"title": "GoJSX — Go + Goja + RSC"}
		},
	})
	app.Register(Route{
		Pattern: "/products",
		Export:  "ProductsPage",
		PropsFunc: func(r *http.Request) map[string]any {
			return map[string]any{"category": r.URL.Query().Get("category")}
		},
	})
	app.Register(Route{
		Pattern: "/user",
		Export:  "UserPage",
		PropsFunc: func(r *http.Request) map[string]any {
			return map[string]any{"userID": r.URL.Query().Get("id")}
		},
	})
	app.Register(Route{
		Pattern: "/about",
		Export:  "AboutPage",
		PropsFunc: func(r *http.Request) map[string]any {
			return map[string]any{"version": "0.1.0"}
		},
	})

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
	tree := flightTree(t, body)
	treeJSON, _ := json.Marshal(tree)
	s := string(treeJSON)
	for _, want := range []string{"Widget Alpha", "Widget Beta", "product-list"} {
		if !strings.Contains(s, want) {
			t.Errorf("ProductsPage missing %q in tree", want)
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
	if arr[1] != "div" {
		t.Errorf("expected 'div' as element type, got %v", arr[1])
	}
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
	if !ok {
		t.Fatalf("expected array root, got %T", tree)
	}
	if len(arr) < 2 || arr[1] != "div" {
		t.Errorf("expected root div element, got %v", arr)
	}
}

func TestFlightTree_UserPage_Structure(t *testing.T) {
	body := rsc(t, "/user")
	tree := flightTree(t, body)
	if !walkFlight(tree, "Jane Doe") {
		t.Error("UserPage tree missing 'Jane Doe'")
	}
	if !walkFlight(tree, "admin") {
		t.Error("UserPage tree missing role 'admin'")
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

	for i := 0; i < n; i++ {
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

	for i := 0; i < n; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}

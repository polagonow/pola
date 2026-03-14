package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"gojsx/build"
	"gojsx/runtime"
)

// Route maps a URL pattern to a Server Component export name and its props factory.
type Route struct {
	Pattern   string
	Export    string // exported function name in the server bundle
	PropsFunc func(r *http.Request) map[string]any
}

// App is the top-level application.
type App struct {
	pool              *runtime.VMPool
	renderer          *runtime.Renderer
	routes            []Route
	publicDir         string
	manifest          runtime.ClientManifest
	importURLs        map[string]string // moduleId → /public/chunk-HASH.js
	clientEntryScript string            // /public/client-entry-[hash].js
}

func main() {
	appDir := "./app"
	publicDir := "./public"

	// ------------------------------------------------------------------
	// 1. Build server + client bundles via esbuild
	// ------------------------------------------------------------------
	bundleResult, err := build.Bundle(build.BundlerConfig{
		AppDir:      appDir,
		OutDir:      publicDir,
		ClientEntry: filepath.Join(appDir, "client-entry.tsx"),
		PolyfillsJS: "./runtime/polyfills.js",
		Pages: []build.PageEntry{
			{File: filepath.Join(appDir, "pages/index.tsx"), Export: "IndexPage"},
			{File: filepath.Join(appDir, "pages/products.tsx"), Export: "ProductsPage"},
			{File: filepath.Join(appDir, "pages/user.tsx"), Export: "UserPage"},
			{File: filepath.Join(appDir, "pages/about.tsx"), Export: "AboutPage"},
		},
		ClientComponents: []string{
			filepath.Join(appDir, "components/Counter.tsx"),
			filepath.Join(appDir, "components/ThemeToggle.tsx"),
		},
	})
	if err != nil {
		log.Fatalf("⚠️  bundle warning: %v", err)
	}

	// ------------------------------------------------------------------
	// 2. Parse client manifest
	// ------------------------------------------------------------------
	manifest, err := runtime.LoadManifest(bundleResult.Manifest)
	if err != nil {
		log.Fatalf("manifest: %v", err)
	}

	// ------------------------------------------------------------------
	// 3. Wire Go → JS bridge
	//    Globals  → bare function calls:  getEnv("KEY")
	//    Context  → ctx object calls:     ctx.getProducts()
	// ------------------------------------------------------------------
	bridge := runtime.BridgeConfig{
		Globals: map[string]runtime.GoFunc{
			"fetchJSON": func(args []runtime.GoValue) (any, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("fetchJSON requires a url argument")
				}
				resp, err := http.Get(args[0].String()) //nolint:gosec
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				var v any
				return v, json.NewDecoder(resp.Body).Decode(&v)
			},
			"getEnv": func(args []runtime.GoValue) (any, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("getEnv requires a key argument")
				}
				// Read from actual env, safe subset only
				key := args[0].String()
				allowed := map[string]bool{"APP_NAME": true, "VERSION": true}
				if !allowed[key] {
					return "", nil
				}
				return key, nil // return key as demo value
			},
		},
		Context: map[string]runtime.GoFunc{
			"getProducts": func(args []runtime.GoValue) (any, error) {
				time.Sleep(10 * time.Second)
				return []map[string]any{
					{"id": 1, "name": "Widget Alpha", "price": 29.99, "stock": 142},
					{"id": 2, "name": "Widget Beta", "price": 49.99, "stock": 37},
					{"id": 3, "name": "Widget Gamma", "price": 9.99, "stock": 891},
					{"id": 4, "name": "Turbo Sprocket", "price": 199.0, "stock": 12},
				}, nil
			},
			"getUser": func(args []runtime.GoValue) (any, error) {
				id := "anonymous"
				if len(args) > 0 {
					id = args[0].String()
				}
				return map[string]any{
					"id":    id,
					"name":  "Jane Doe",
					"email": "jane@example.com",
					"role":  "admin",
				}, nil
			},
			"query": func(args []runtime.GoValue) (any, error) {
				// Placeholder for real DB access
				return []any{}, nil
			},
		},
	}

	// ------------------------------------------------------------------
	// 4. Boot VM pool
	// ------------------------------------------------------------------
	serverJS := bundleResult.ServerBundle
	pool, err := runtime.NewVMPool(serverJS, bridge)
	if err != nil {
		log.Fatalf("vm pool: %v", err)
	}

	app := &App{
		pool:              pool,
		renderer:          runtime.NewRenderer(pool, manifest),
		publicDir:         publicDir,
		manifest:          manifest,
		importURLs:        bundleResult.ImportURLs,
		clientEntryScript: bundleResult.ClientEntryOutput,
	}

	// ------------------------------------------------------------------
	// 5. Manual route registration
	// ------------------------------------------------------------------
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

	// ------------------------------------------------------------------
	// 6. HTTP handlers
	// ------------------------------------------------------------------
	// Static assets
	http.Handle("/public/", http.StripPrefix("/public/",
		http.FileServer(http.Dir(publicDir))))

	// All page routes: HTML shell or RSC Flight based on Content-Type header.
	http.HandleFunc("/", app.handleRoute)

	addr := ":3000"
	log.Printf("🚀 GoJSX running → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func (a *App) Register(r Route) {
	a.routes = append(a.routes, r)
}

func (a *App) match(path string) *Route {
	for i := range a.routes {
		if a.routes[i].Pattern == path {
			return &a.routes[i]
		}
	}
	return &a.routes[0]
}

// handleRoute serves all page routes. When the request carries
// Content-Type: text/x-component it returns the RSC Flight stream;
// otherwise it returns the HTML shell that bootstraps the client.
func (a *App) handleRoute(w http.ResponseWriter, r *http.Request) {
	route := a.match(r.URL.Path)
	if route == nil {
		http.NotFound(w, r)
		return
	}

	if r.Header.Get("Content-Type") == "text/x-component" {
		w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
		fw := runtime.NewFlightWriter(w)
		vm := a.pool.Acquire()
		defer a.pool.Release(vm)
		if err := a.renderer.Render(fw, vm, runtime.RenderOptions{
			ExportName:     route.Export,
			Props:          route.PropsFunc(r),
			RequestContext: requestCtx(r),
		}); err != nil {
			log.Printf("rsc %s: %v", r.URL.Path, err)
			fmt.Fprintf(w, `<div class="rsc-err">%s</div>`, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlShell(buildImportMap(a.importURLs)))
	fmt.Fprint(w, htmlClose(a.clientEntryScript))
}

func requestCtx(r *http.Request) map[string]any {
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	return map[string]any{
		"url":     r.URL.String(),
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"method":  r.Method,
		"headers": headers,
	}
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ---- HTML shell templates ------------------------------------------------

// buildImportMap generates a <script type="importmap"> block that maps RSC
// module IDs (e.g. "components/ThemeToggle") to the hashed client bundle URLs
// emitted by esbuild (e.g. "/public/ThemeToggle-XYZ.js"). This lets the
// react-server-dom-esm client import() those IDs directly in the browser.
func buildImportMap(importURLs map[string]string) string {
	if len(importURLs) == 0 {
		return ""
	}
	payload, err := json.Marshal(map[string]any{"imports": importURLs})
	if err != nil {
		return ""
	}
	return fmt.Sprintf(`<script type="importmap">%s</script>`, payload)
}

func htmlShell(importMap string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>GoJSX</title>
  <style>
    *,*::before,*::after{box-sizing:border-box}
    body{font-family:system-ui,sans-serif;margin:0;padding:2rem;background:#f8f8f8;color:#111}
    .page{max-width:860px;margin:0 auto}
    h1{font-size:2rem;margin:0 0 .5rem}
    h2{font-size:1.2rem;margin:1.5rem 0 .5rem;color:#444}
    code{background:#eee;padding:.1em .35em;border-radius:4px;font-size:.9em}
    nav{display:flex;gap:1rem;margin-bottom:1.5rem;padding-bottom:1rem;border-bottom:1px solid #e0e0e0}
    nav a{color:#0070f3;text-decoration:none;font-weight:500}
    nav a:hover{text-decoration:underline}
    .product-list{display:grid;gap:.5rem;margin-top:.5rem}
    .product{background:#fff;padding:.75rem 1rem;border-radius:8px;border:1px solid #e5e5e5;
             display:flex;align-items:center;gap:.75rem}
    .product strong{flex:1}
    .product .price{color:#0070f3;font-weight:600}
    .product small{color:#999}
    dl{display:grid;grid-template-columns:max-content 1fr;gap:.25rem 1.5rem;margin-top:.5rem}
    dt{font-weight:600;color:#555}
    #root{padding-top:.5rem}
    .rsc-err{color:#c0392b;background:#fdecea;padding:.75rem 1rem;border-radius:8px}
  </style>
</head>
<body>
` + importMap + `
  <nav>
    <a href="/">Home</a>
    <a href="/products">Products</a>
    <a href="/user?id=42">Profile</a>
    <a href="/about">About</a>
  </nav>
  <div id="root"></div>
`
}

func htmlClose(clientScript string) string {
	// Fall back to the legacy manual renderer if the esbuild entry was not found
	if clientScript == "" {
		clientScript = "/public/rsc-client.js"
	}
	return fmt.Sprintf(`
  <script type="module" src="%s"></script>
</body>
</html>`, clientScript)
}

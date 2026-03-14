package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gojsx/build"
	ghtml "gojsx/html"
	"gojsx/runtime"
)

// ssrStreaming controls whether the HTML path streams RSC via a second
// client fetch (true, default) or buffers and inlines flight data in the
// HTML response (false). Streaming supports Suspense; inlining saves a round-trip.
var ssrStreaming = os.Getenv("SSR_STREAMING") != "false"

const (
	defaultPublicDir = "./public"
	defaultPublicURL = "/public"
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
	clientEntryScript string            // /public/client-[hash].js
}

func main() {
	appDir := "./app"

	// ------------------------------------------------------------------
	// 1. Build server + client bundles via esbuild
	// ------------------------------------------------------------------
	bundleResult, err := build.Bundle(build.BundlerConfig{
		AppDir:        appDir,
		OutDir:        defaultPublicDir + "/assets",
		AssetsURLPath: defaultPublicURL + "/assets",
		ClientEntry:   filepath.Join(appDir, "_client.tsx"),
		PolyfillsJS:   "./runtime/polyfills.js",
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
			"fetchJSON": func(args []any) (any, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("fetchJSON requires a url argument")
				}
				resp, err := http.Get(fmt.Sprintf("%v", args[0])) //nolint:gosec
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				var v any
				return v, json.NewDecoder(resp.Body).Decode(&v)
			},
			"getEnv": func(args []any) (any, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("getEnv requires a key argument")
				}
				// Read from actual env, safe subset only
				key := fmt.Sprintf("%v", args[0])
				allowed := map[string]bool{"APP_NAME": true, "VERSION": true}
				if !allowed[key] {
					return "", nil
				}
				return key, nil // return key as demo value
			},
		},
		Context: map[string]runtime.GoFunc{
			"getProducts": func(args []any) (any, error) {
				time.Sleep(500 * time.Millisecond)
				return []map[string]any{
					{"id": 1, "name": "Widget Alpha", "price": 29.99, "stock": 142},
					{"id": 2, "name": "Widget Beta", "price": 49.99, "stock": 37},
					{"id": 3, "name": "Widget Gamma", "price": 9.99, "stock": 891},
					{"id": 4, "name": "Turbo Sprocket", "price": 199.0, "stock": 12},
				}, nil
			},
			"getUser": func(args []any) (any, error) {
				id := "anonymous"
				if len(args) > 0 {
					id = fmt.Sprintf("%v", args[0])
				}
				return map[string]any{
					"id":    id,
					"name":  "Jane Doe",
					"email": "jane@example.com",
					"role":  "admin",
				}, nil
			},
			"query": func(args []any) (any, error) {
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
		publicDir:         defaultPublicDir,
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
	http.Handle(defaultPublicURL+"/", http.StripPrefix(defaultPublicURL+"/",
		http.FileServer(http.Dir(defaultPublicDir))))

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
	fmt.Fprint(w, ghtml.Open(ghtml.Params{ImportURLs: a.importURLs}))

	if ssrStreaming {
		// Streaming mode (default): serve empty shell; client fetches RSC separately.
		// Supports Suspense — async chunks stream in as server components resolve.
	} else {
		// Inline mode: buffer the full RSC flight payload and embed it in the HTML.
		// Eliminates the second fetch but requires the full render to complete before
		// any bytes are sent — no Suspense streaming.
		var buf bytes.Buffer
		bfw := runtime.NewFlightWriterBuffer(&buf)
		vm := a.pool.Acquire()
		defer a.pool.Release(vm)
		if err := a.renderer.Render(bfw, vm, runtime.RenderOptions{
			ExportName:     route.Export,
			Props:          route.PropsFunc(r),
			RequestContext: requestCtx(r),
		}); err != nil {
			log.Printf("html render %s: %v", r.URL.Path, err)
		}
		flightJSON, _ := json.Marshal(buf.String())
		fmt.Fprintf(w, "<script>self.__flight_data=%s</script>\n", flightJSON)
	}

	fmt.Fprint(w, ghtml.Close(ghtml.Params{ClientScript: a.clientEntryScript}))
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


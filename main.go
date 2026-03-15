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

// Route maps a URL pattern to a Server Component export name.
// Pattern supports dynamic segments: "/products/:id".
// Export is the JS key in __pages__ (e.g. "Products").
type Route struct {
	Pattern string
	Export  string
}

// App is the top-level application.
type App struct {
	pool                *runtime.VMPool
	renderer            *runtime.Renderer
	routes              []Route
	publicDir           string
	manifest            runtime.ClientManifest
	importURLs          map[string]string // moduleId → /public/chunk-HASH.js
	clientEntryScript   string            // /public/client-[hash].js
	globalNotFoundExport string           // "GlobalNotFound" or ""
}

func main() {
	appDir := "./ui"

	// ------------------------------------------------------------------
	// 1. Build server + client bundles via esbuild
	// ------------------------------------------------------------------
	pages, err := build.DiscoverPages(appDir)
	if err != nil {
		log.Fatalf("discover pages: %v", err)
	}
	gc, err := build.DiscoverGlobalComponents(appDir)
	if err != nil {
		log.Fatalf("discover global components: %v", err)
	}
	clientComponents, err := build.DiscoverClientComponents(appDir)
	if err != nil {
		log.Fatalf("discover client components: %v", err)
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
		OutDir:             defaultPublicDir + "/assets",
		AssetsURLPath:      defaultPublicURL + "/assets",
		ClientEntry:        filepath.Join(appDir, "_client.tsx"),
		Pages:              pages,
		ClientComponents:   clientComponents,
		GlobalNotFoundPath: gc.NotFoundPath,
		GlobalErrorPath:    gc.ErrorPath,
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
	productCatalog := []map[string]any{
		{"id": 1, "name": "Widget Alpha", "price": 29.99, "stock": 142},
		{"id": 2, "name": "Widget Beta", "price": 49.99, "stock": 37},
		{"id": 3, "name": "Widget Gamma", "price": 9.99, "stock": 891},
		{"id": 4, "name": "Turbo Sprocket", "price": 199.0, "stock": 12},
	}

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
				key := fmt.Sprintf("%v", args[0])
				allowed := map[string]bool{"APP_NAME": true, "VERSION": true}
				if !allowed[key] {
					return "", nil
				}
				return key, nil
			},
		},
		Context: map[string]runtime.GoFunc{
			"getProducts": func(args []any) (any, error) {
				time.Sleep(500 * time.Millisecond)
				return productCatalog, nil
			},
			"getProduct": func(args []any) (any, error) {
				time.Sleep(500 * time.Millisecond)
				id := ""
				if len(args) > 0 {
					id = fmt.Sprintf("%v", args[0])
				}
				for _, p := range productCatalog {
					if fmt.Sprintf("%v", p["id"]) == id {
						return p, nil
					}
				}
				return nil, fmt.Errorf("product %q not found", id)
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
				return []any{}, nil
			},
		},
	}

	// ------------------------------------------------------------------
	// 4. Boot VM pool
	// ------------------------------------------------------------------
	serverJS, err := os.ReadFile(bundleResult.ServerBundlePath)
	if err != nil {
		log.Fatalf("read server bundle: %v", err)
	}
	pool, err := runtime.NewVMPool(string(serverJS), bridge)
	if err != nil {
		log.Fatalf("vm pool: %v", err)
	}

	globalNotFoundExport := ""
	if gc.NotFoundPath != "" {
		globalNotFoundExport = "GlobalNotFound"
	}
	app := &App{
		pool:                 pool,
		renderer:             runtime.NewRenderer(pool, manifest),
		publicDir:            defaultPublicDir,
		manifest:             manifest,
		importURLs:           bundleResult.ImportURLs,
		clientEntryScript:    bundleResult.ClientEntryOutput,
		globalNotFoundExport: globalNotFoundExport,
	}

	// ------------------------------------------------------------------
	// 5. Auto-register routes from discovered pages
	// ------------------------------------------------------------------
	for _, p := range pages {
		app.routes = append(app.routes, Route{
			Pattern: build.RoutePattern(appDir, p.PageComponentPath),
			Export:  build.PageAlias(p),
		})
	}

	// ------------------------------------------------------------------
	// 6. HTTP handlers
	// ------------------------------------------------------------------
	http.Handle(defaultPublicURL+"/", http.StripPrefix(defaultPublicURL+"/",
		http.FileServer(http.Dir(defaultPublicDir))))
	http.HandleFunc("/", app.handleRoute)

	addr := ":3000"
	log.Printf("🚀 GoJSX running → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// matchPattern matches a URL path against a route pattern.
// Dynamic segments (":name") capture into the returned map.
// Returns (params, true) on match, (nil, false) otherwise.
func matchPattern(pattern, path string) (map[string]string, bool) {
	pp := strings.Split(pattern, "/")
	rp := strings.Split(path, "/")
	if len(pp) != len(rp) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range pp {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = rp[i]
		} else if seg != rp[i] {
			return nil, false
		}
	}
	return params, true
}

func (a *App) match(path string) (*Route, map[string]string) {
	for i := range a.routes {
		if params, ok := matchPattern(a.routes[i].Pattern, path); ok {
			return &a.routes[i], params
		}
	}
	return nil, nil
}

// buildPageProps constructs the standard PageProps passed to every page component:
//
//	{ params: { <path segments> }, searchParams: { <query string> } }
func buildPageProps(r *http.Request, params map[string]string) map[string]any {
	searchParams := map[string]any{}
	for k, vs := range r.URL.Query() {
		if len(vs) == 1 {
			searchParams[k] = vs[0]
		} else {
			searchParams[k] = vs
		}
	}
	p := map[string]any{}
	for k, v := range params {
		p[k] = v
	}
	return map[string]any{"params": p, "searchParams": searchParams}
}

// handleRoute serves all page routes. When the request carries
// Content-Type: text/x-component it returns the RSC Flight stream;
// otherwise it returns the HTML shell that bootstraps the client.
func (a *App) handleRoute(w http.ResponseWriter, r *http.Request) {
	route, params := a.match(r.URL.Path)
	if route == nil {
		if a.globalNotFoundExport != "" {
			if r.Header.Get("Content-Type") == "text/x-component" {
				w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				fw := runtime.NewFlightWriter(w)
				vm := a.pool.Acquire()
				defer a.pool.Release(vm)
				if err := a.renderer.Render(fw, vm, runtime.RenderOptions{
					ExportName: a.globalNotFoundExport,
					Props:      map[string]any{},
				}); err != nil {
					log.Printf("rsc 404: %v", err)
				}
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, ghtml.Render(ghtml.Params{
					ImportURLs:   a.importURLs,
					ClientScript: a.clientEntryScript,
				}))
			}
		} else {
			http.NotFound(w, r)
		}
		return
	}

	props := buildPageProps(r, params)

	if r.Header.Get("Content-Type") == "text/x-component" {
		w.Header().Set("Content-Type", "text/x-component; charset=utf-8")
		fw := runtime.NewFlightWriter(w)
		vm := a.pool.Acquire()
		defer a.pool.Release(vm)
		if err := a.renderer.Render(fw, vm, runtime.RenderOptions{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: requestCtx(r),
		}); err != nil {
			log.Printf("rsc %s: %v", r.URL.Path, err)
			fmt.Fprintf(w, `<div class="rsc-err">%s</div>`, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	htmlParams := ghtml.Params{
		ImportURLs:   a.importURLs,
		ClientScript: a.clientEntryScript,
	}

	if !ssrStreaming {
		var buf bytes.Buffer
		bfw := runtime.NewFlightWriterBuffer(&buf)
		vm := a.pool.Acquire()
		defer a.pool.Release(vm)
		if err := a.renderer.Render(bfw, vm, runtime.RenderOptions{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: requestCtx(r),
		}); err != nil {
			log.Printf("html render %s: %v", r.URL.Path, err)
		}
		flightJSON, _ := json.Marshal(buf.String())
		htmlParams.Scripts = append(htmlParams.Scripts, "self.__flight_data="+string(flightJSON))
	}

	fmt.Fprint(w, ghtml.Render(htmlParams))
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

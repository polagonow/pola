package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gojsx/build"
	"gojsx/server"
	"gojsx/runtime"
)

const (
	defaultPublicDir = "../public"
	defaultPublicURL = "/public"
)

func main() {
	appDir := "../ui"

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
	app := &server.App{
		Pool:                 pool,
		Renderer:             runtime.NewRenderer(pool, manifest),
		PublicDir:            defaultPublicDir,
		Manifest:             manifest,
		ImportURLs:           bundleResult.ImportURLs,
		ClientEntryScript:    bundleResult.ClientEntryOutput,
		GlobalNotFoundExport: globalNotFoundExport,
	}

	// ------------------------------------------------------------------
	// 5. Auto-register routes from discovered pages
	// ------------------------------------------------------------------
	for _, p := range pages {
		app.Routes = append(app.Routes, server.Route{
			Pattern: build.RoutePattern(appDir, p.PageComponentPath),
			Export:  build.PageAlias(p),
		})
	}

	// ------------------------------------------------------------------
	// 6. HTTP handlers
	// ------------------------------------------------------------------
	http.Handle(defaultPublicURL+"/", http.StripPrefix(defaultPublicURL+"/",
		http.FileServer(http.Dir(defaultPublicDir))))
	http.HandleFunc("/", app.HandleRoute)

	addr := ":3000"
	log.Printf("🚀 GoJSX running → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

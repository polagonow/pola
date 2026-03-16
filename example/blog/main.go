package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	// Blank imports register default implementations (esbuild, Goja, React HTML shell, disk assets).
	_ "gojsx/bundler"
	"gojsx/framework"
	_ "gojsx/framework/assets/disk"
	"gojsx/framework/contract"
	_ "gojsx/render"
	_ "gojsx/vm"
)

func main() {
	// ------------------------------------------------------------------
	// 1. Wire Go → JS bridge (application-specific data access)
	// ------------------------------------------------------------------
	posts := []map[string]any{
		{
			"id": 1, "slug": "go-react-ssr",
			"title":   "Building SSR with Go and React",
			"excerpt": "How to run React Server Components inside a Go process using Goja.",
			"author":  "Jane Doe", "date": "2024-01-15", "readTime": 5,
			"tags": []any{"go", "react", "ssr"},
		},
		{
			"id": 2, "slug": "rsc-deep-dive",
			"title":   "React Server Components Deep Dive",
			"excerpt": "Understanding the Flight wire protocol and how RSC trees serialize.",
			"author":  "Jane Doe", "date": "2024-02-03", "readTime": 8,
			"tags": []any{"react", "rsc", "performance"},
		},
		{
			"id": 3, "slug": "goja-vm-internals",
			"title":   "Goja VM Internals",
			"excerpt": "A tour through the event loop, promise scheduling, and Go↔JS bridging.",
			"author":  "Jane Doe", "date": "2024-03-10", "readTime": 12,
			"tags": []any{"go", "javascript", "vm"},
		},
	}

	revisions := map[string][]map[string]any{
		"go-react-ssr": {
			{"rev": "v3", "date": "2024-01-15", "summary": "Published — added Suspense streaming section."},
			{"rev": "v2", "date": "2024-01-10", "summary": "Draft — expanded esbuild two-pass explanation."},
			{"rev": "v1", "date": "2024-01-05", "summary": "Initial draft — skeleton outline only."},
		},
		"rsc-deep-dive": {
			{"rev": "v2", "date": "2024-02-03", "summary": "Published — Flight wire-format diagrams added."},
			{"rev": "v1", "date": "2024-01-28", "summary": "Initial draft — protocol walkthrough."},
		},
		"goja-vm-internals": {
			{"rev": "v1", "date": "2024-03-10", "summary": "Published — first and only revision."},
		},
	}

	projects := []map[string]any{
		{
			"id": "1", "title": "GoJSX",
			"description": "Go-powered React SSR framework with Flight protocol.",
			"tech":        []any{"Go", "React", "TypeScript", "esbuild"},
			"stars":       142, "status": "active",
		},
		{
			"id": "2", "title": "GojaBridge",
			"description": "Type-safe Go ↔ JS function bridge for Goja.",
			"tech":        []any{"Go", "Goja"},
			"stars":       38, "status": "stable",
		},
		{
			"id": "3", "title": "FlightDecode",
			"description": "Pure-Go React Flight wire-format decoder.",
			"tech":        []any{"Go", "React"},
			"stars":       21, "status": "beta",
		},
	}

	globalBridge := contract.BridgeConfig{
		Globals: map[string]contract.GoFunc{
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
		Context: map[string]contract.GoFunc{
			"getPosts": func(_ []any) (any, error) {
				time.Sleep(200 * time.Millisecond)
				return posts, nil
			},
			"getPost": func(args []any) (any, error) {
				time.Sleep(150 * time.Millisecond)
				slug := ""
				if len(args) > 0 {
					slug = fmt.Sprintf("%v", args[0])
				}
				for _, p := range posts {
					if p["slug"] == slug {
						return p, nil
					}
				}
				return nil, fmt.Errorf("post %q not found", slug)
			},
			"getProjects": func(_ []any) (any, error) {
				time.Sleep(200 * time.Millisecond)
				return projects, nil
			},
			"getProject": func(args []any) (any, error) {
				time.Sleep(150 * time.Millisecond)
				id := ""
				if len(args) > 0 {
					id = fmt.Sprintf("%v", args[0])
				}
				for _, p := range projects {
					if p["id"] == id {
						return p, nil
					}
				}
				return nil, fmt.Errorf("project %q not found", id)
			},
			"getRevisions": func(args []any) (any, error) {
				slug := ""
				if len(args) > 0 {
					slug = fmt.Sprintf("%v", args[0])
				}
				if revs, ok := revisions[slug]; ok {
					return revs, nil
				}
				return nil, fmt.Errorf("no revisions for post %q", slug)
			},
			"getRevision": func(args []any) (any, error) {
				slug, rev := "", ""
				if len(args) > 0 {
					slug = fmt.Sprintf("%v", args[0])
				}
				if len(args) > 1 {
					rev = fmt.Sprintf("%v", args[1])
				}
				for _, r := range revisions[slug] {
					if r["rev"] == rev {
						return r, nil
					}
				}
				return nil, fmt.Errorf("revision %q not found for post %q", rev, slug)
			},
			"getProfile": func(_ []any) (any, error) {
				return map[string]any{
					"id":      "1",
					"name":    "Jane Doe",
					"email":   "jane@example.com",
					"role":    "Senior Engineer",
					"bio":     "Building dev tools with Go and React.",
					"github":  "janedoe",
					"website": "https://janedoe.dev",
				}, nil
			},
			"triggerError": func(args []any) (any, error) {
				msg := "Forced error for testing"
				if len(args) > 0 {
					if s := fmt.Sprintf("%v", args[0]); s != "" {
						msg = s
					}
				}
				return nil, errors.New(msg)
			},
		},
	}

	// Per-route JSI bridges — each route only exposes the functions it needs.
	postsBridge := &contract.BridgeConfig{Context: map[string]contract.GoFunc{
		"getPosts":     globalBridge.Context["getPosts"],
		"getPost":      globalBridge.Context["getPost"],
		"getRevisions": globalBridge.Context["getRevisions"],
		"getRevision":  globalBridge.Context["getRevision"],
		"triggerError": globalBridge.Context["triggerError"],
	}}
	projectsBridge := &contract.BridgeConfig{Context: map[string]contract.GoFunc{
		"getProjects":  globalBridge.Context["getProjects"],
		"getProject":   globalBridge.Context["getProject"],
		"triggerError": globalBridge.Context["triggerError"],
	}}
	profileBridge := &contract.BridgeConfig{Context: map[string]contract.GoFunc{
		"getProfile":   globalBridge.Context["getProfile"],
		"triggerError": globalBridge.Context["triggerError"],
	}}
	homeBridge := &contract.BridgeConfig{Context: map[string]contract.GoFunc{
		"getPosts":     globalBridge.Context["getPosts"],
		"getProjects":  globalBridge.Context["getProjects"],
		"triggerError": globalBridge.Context["triggerError"],
	}}

	// ------------------------------------------------------------------
	// 2. Build + wire the app via framework.Config
	//    Discovery, bundling, VM pool, routing are all handled automatically.
	// ------------------------------------------------------------------
	cfg := &framework.Config{
		AppDir:       "../../ui/apps/blog",
		Dev:          true,
		GlobalBridge: globalBridge,
	}
	app, err := cfg.Build()
	if err != nil {
		log.Fatalf("build: %v", err)
	}

	// ------------------------------------------------------------------
	// 3. Apply per-route bridge overrides
	// ------------------------------------------------------------------
	routeBridges := map[string]*contract.BridgeConfig{
		"/":                           homeBridge,
		"/posts":                      postsBridge,
		"/posts/:slug":                postsBridge,
		"/posts/:slug/revisions":      postsBridge,
		"/posts/:slug/revisions/:rev": postsBridge,
		"/projects":                   projectsBridge,
		"/projects/:id":               projectsBridge,
		"/profile":                    profileBridge,
	}
	for i := range app.Artifacts().Routes {
		if b := routeBridges[app.Artifacts().Routes[i].Pattern]; b != nil {
			app.Artifacts().Routes[i].Bridge = b
		}
	}

	// ------------------------------------------------------------------
	// 4. Start server
	// ------------------------------------------------------------------
	addr := ":3000"
	log.Printf("🚀 GoJSX running → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, app.Handler()))
}

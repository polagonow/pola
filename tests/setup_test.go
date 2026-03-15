package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	build "gojsx/bundler"
	"gojsx/framework"
	"gojsx/framework/contract"
	renderreact "gojsx/render/react"
	nextjs "gojsx/render/react/discovery/nextjs"
	"gojsx/server"
	vmgoja "gojsx/vm/goja"
)

var (
	_testApp *server.App
	_testErr error
)

func init() {
	_testApp, _testErr = newTestApp()
}

func newTestApp() (*server.App, error) {
	appDir := "../ui"

	pages, err := nextjs.DiscoverPages(appDir)
	if err != nil {
		return nil, fmt.Errorf("discover pages: %w", err)
	}
	gc, err := nextjs.DiscoverGlobalComponents(appDir)
	if err != nil {
		return nil, fmt.Errorf("discover global components: %w", err)
	}
	clientComponents, err := nextjs.DiscoverClientComponents(appDir)
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

	posts := []map[string]any{
		{"id": 1, "slug": "go-react-ssr", "title": "Building SSR with Go and React",
			"excerpt": "How to run React Server Components inside a Go process using Goja.",
			"author":  "Jane Doe", "date": "2024-01-15", "readTime": 5,
			"tags": []any{"go", "react", "ssr"}},
		{"id": 2, "slug": "rsc-deep-dive", "title": "React Server Components Deep Dive",
			"excerpt": "Understanding the Flight wire protocol and how RSC trees serialize.",
			"author":  "Jane Doe", "date": "2024-02-03", "readTime": 8,
			"tags": []any{"react", "rsc", "performance"}},
		{"id": 3, "slug": "goja-vm-internals", "title": "Goja VM Internals",
			"excerpt": "A tour through the event loop, promise scheduling, and Go↔JS bridging.",
			"author":  "Jane Doe", "date": "2024-03-10", "readTime": 12,
			"tags": []any{"go", "javascript", "vm"}},
	}
	projects := []map[string]any{
		{"id": "1", "title": "GoJSX", "description": "Go-powered React SSR framework.",
			"tech": []any{"Go", "React", "TypeScript", "esbuild"}, "stars": 142, "status": "active"},
		{"id": "2", "title": "GojaBridge", "description": "Type-safe Go ↔ JS bridge.",
			"tech": []any{"Go", "Goja"}, "stars": 38, "status": "stable"},
		{"id": "3", "title": "FlightDecode", "description": "Pure-Go Flight wire-format decoder.",
			"tech": []any{"Go", "React"}, "stars": 21, "status": "beta"},
	}

	bridge := contract.BridgeConfig{
		Context: map[string]contract.GoFunc{
			"getPosts": func(_ []any) (any, error) { return posts, nil },
			"getPost": func(args []any) (any, error) {
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
			"getProjects": func(_ []any) (any, error) { return projects, nil },
			"getProject": func(args []any) (any, error) {
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
					"id": "1", "name": "Jane Doe", "email": "jane@example.com",
					"role": "Senior Engineer", "bio": "Building dev tools.",
					"github": "janedoe", "website": "https://janedoe.dev",
				}, nil
			},
			"triggerError": func(args []any) (any, error) {
				msg := "Forced error for testing"
				if len(args) > 0 {
					if s := fmt.Sprintf("%v", args[0]); s != "" {
						msg = s
					}
				}
				return nil, fmt.Errorf("%s", msg)
			},
		},
	}

	gen := nextjs.ReactRSCEntryGenerator{}
	entryContent, err := gen.Generate(framework.EntryGenConfig{
		AppDir:             appDir,
		Pages:              pages,
		GlobalNotFoundPath: gc.NotFoundPath,
		GlobalErrorPath:    gc.ErrorPath,
	})
	if err != nil {
		return nil, fmt.Errorf("entry generate: %w", err)
	}

	bundleResult, err := build.Bundle(build.BundlerConfig{
		AppDir:                 appDir,
		OutDir:                 "../public/assets",
		ClientEntry:            "../ui/_client.tsx",
		ClientComponents:       clientComponents,
		ServerEntryContent:     entryContent,
		ServerBundleConditions: gen.BundleConditions(),
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
	}

	pool, err := vmgoja.NewVMPool(string(bundleResult.ServerBundle), bridge)
	if err != nil {
		return nil, fmt.Errorf("vm pool: %w", err)
	}
	manifest, err := renderreact.LoadManifest(bundleResult.Manifest)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	globalNotFoundExport := ""
	if gc.NotFoundPath != "" {
		globalNotFoundExport = "GlobalNotFound"
	}
	app := &server.App{
		Pool:                 pool,
		Renderer:             renderreact.NewRenderer(pool, manifest),
		ClientEntryScript:    bundleResult.ClientEntryOutput,
		ImportURLs:           bundleResult.ImportURLs,
		GlobalNotFoundExport: globalNotFoundExport,
	}
	for _, p := range pages {
		app.Routes = append(app.Routes, server.Route{
			Pattern: nextjs.RoutePattern(appDir, p.PageComponentPath),
			Export:  nextjs.PageAlias(p),
		})
	}
	return app, nil
}

func requireApp(t *testing.T) *server.App {
	t.Helper()
	if _testErr != nil {
		t.Fatalf("app init failed: %v", _testErr)
	}
	return _testApp
}

// rsc makes an RSC request (Content-Type: text/x-component) and returns the body.
func rsc(t *testing.T, path string) string {
	t.Helper()
	app := requireApp(t)
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("RSC GET %s → %d", path, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// page makes a normal HTML request and returns the body.
func page(t *testing.T, path string) string {
	t.Helper()
	app := requireApp(t)
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s → %d", path, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// flightTree parses the root Flight row (0:) as JSON.
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

func flightContains(body, substr string) bool { return strings.Contains(body, substr) }

package tests

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
	"gojsx/server"
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
			"tags":    []any{"go", "react", "ssr"}},
		{"id": 2, "slug": "rsc-deep-dive", "title": "React Server Components Deep Dive",
			"excerpt": "Understanding the Flight wire protocol and how RSC trees serialize.",
			"author":  "Jane Doe", "date": "2024-02-03", "readTime": 8,
			"tags":    []any{"react", "rsc", "performance"}},
		{"id": 3, "slug": "goja-vm-internals", "title": "Goja VM Internals",
			"excerpt": "A tour through the event loop, promise scheduling, and Go↔JS bridging.",
			"author":  "Jane Doe", "date": "2024-03-10", "readTime": 12,
			"tags":    []any{"go", "javascript", "vm"}},
	}
	projects := []map[string]any{
		{"id": "1", "title": "GoJSX", "description": "Go-powered React SSR framework.",
			"tech": []any{"Go", "React", "TypeScript", "esbuild"}, "stars": 142, "status": "active"},
		{"id": "2", "title": "GojaBridge", "description": "Type-safe Go ↔ JS bridge.",
			"tech": []any{"Go", "Goja"}, "stars": 38, "status": "stable"},
		{"id": "3", "title": "FlightDecode", "description": "Pure-Go Flight wire-format decoder.",
			"tech": []any{"Go", "React"}, "stars": 21, "status": "beta"},
	}

	bridge := runtime.BridgeConfig{
		Context: map[string]runtime.GoFunc{
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
		},
	}

	bundleResult, err := build.Bundle(build.BundlerConfig{
		AppDir:             appDir,
		OutDir:             "../public/assets",
		ClientEntry:        "../ui/_client.tsx",
		Pages:              pages,
		ClientComponents:   clientComponents,
		GlobalNotFoundPath: gc.NotFoundPath,
		GlobalErrorPath:    gc.ErrorPath,
	})
	if err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
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
	app := &server.App{
		Pool:                 pool,
		Renderer:             runtime.NewRenderer(pool, manifest),
		ClientEntryScript:    bundleResult.ClientEntryOutput,
		ImportURLs:           bundleResult.ImportURLs,
		GlobalNotFoundExport: globalNotFoundExport,
	}
	for _, p := range pages {
		app.Routes = append(app.Routes, server.Route{
			Pattern: build.RoutePattern(appDir, p.PageComponentPath),
			Export:  build.PageAlias(p),
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

// ─── HTML shell tests ──────────────────────────────────────────────────────────

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

func TestHTMLShell_HasNavLinks(t *testing.T) {
	body := page(t, "/")
	// Nav links are rendered by the client Nav component into #nav-root.
	if !strings.Contains(body, `id="nav-root"`) {
		t.Error("HTML shell missing nav mount point #nav-root")
	}
}

func TestHTMLShell_ContentType(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	ct := w.Result().Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %q", ct)
	}
}

func TestHTMLShell_404(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Result().StatusCode)
	}
}

// ─── RSC tests ─────────────────────────────────────────────────────────────────

func TestRSC_ContentType(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	ct := w.Result().Header.Get("Content-Type")
	if !strings.Contains(ct, "text/x-component") {
		t.Errorf("expected text/x-component, got %q", ct)
	}
}

func TestRSC_HomePage(t *testing.T) {
	body := rsc(t, "/")
	// "DevBlog" is split across a <span> in the JSX, so check the two halves separately.
	for _, want := range []string{"Dev", "Blog", "Recent posts", "Projects"} {
		if !flightContains(body, want) {
			t.Errorf("HomePage Flight missing %q", want)
		}
	}
}

func TestRSC_HomePage_HasPosts(t *testing.T) {
	body := rsc(t, "/")
	for _, want := range []string{"Building SSR with Go and React", "React Server Components Deep Dive"} {
		if !flightContains(body, want) {
			t.Errorf("HomePage missing post %q", want)
		}
	}
}

func TestRSC_PostsPage(t *testing.T) {
	body := rsc(t, "/posts")
	for _, want := range []string{"Building SSR with Go and React", "Goja VM Internals", "go-react-ssr"} {
		if !flightContains(body, want) {
			t.Errorf("PostsPage missing %q", want)
		}
	}
}

func TestRSC_PostsPage_TagFilter(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/posts?tag=go", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("RSC GET /posts?tag=go → %d", w.Result().StatusCode)
	}
	body, _ := io.ReadAll(w.Result().Body)
	if !flightContains(string(body), "go") {
		t.Error("tag filter result missing 'go'")
	}
}

func TestRSC_PostDetailPage(t *testing.T) {
	body := rsc(t, "/posts/go-react-ssr")
	for _, want := range []string{"Building SSR with Go and React", "Jane Doe"} {
		if !flightContains(body, want) {
			t.Errorf("PostDetail missing %q", want)
		}
	}
}

func TestRSC_PostDetailPage_NotFound(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/posts/nonexistent-slug", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
		t.Errorf("expected Flight error row for missing post, got: %s", string(body)[:min(len(string(body)), 200)])
	}
}

func TestRSC_ProjectsPage(t *testing.T) {
	body := rsc(t, "/projects")
	for _, want := range []string{"GoJSX", "GojaBridge", "FlightDecode"} {
		if !flightContains(body, want) {
			t.Errorf("ProjectsPage missing %q", want)
		}
	}
}

func TestRSC_ProjectDetailPage(t *testing.T) {
	body := rsc(t, "/projects/1")
	for _, want := range []string{"GoJSX", "142", "active"} {
		if !flightContains(body, want) {
			t.Errorf("ProjectDetail missing %q", want)
		}
	}
}

func TestRSC_ProjectDetailPage_NotFound(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/projects/999", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
		t.Errorf("expected Flight error row for missing project, got: %s", string(body)[:min(len(string(body)), 200)])
	}
}

func TestRSC_AboutPage(t *testing.T) {
	body := rsc(t, "/about")
	for _, want := range []string{"About", "GoJSX", "Goja"} {
		if !flightContains(body, want) {
			t.Errorf("AboutPage missing %q", want)
		}
	}
}

func TestRSC_ProfilePage(t *testing.T) {
	body := rsc(t, "/profile")
	for _, want := range []string{"Jane Doe", "jane@example.com", "Senior Engineer"} {
		if !flightContains(body, want) {
			t.Errorf("ProfilePage missing %q", want)
		}
	}
}

func TestRSC_ProfilePage_SearchParams(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/profile?id=1", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	if !flightContains(string(body), "Jane Doe") {
		t.Error("ProfilePage with searchParams missing 'Jane Doe'")
	}
}

func TestRSC_404(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown route, got %d", w.Result().StatusCode)
	}
}

// ─── Flight tree structure ─────────────────────────────────────────────────────

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

func TestFlightTree_Root(t *testing.T) {
	body := rsc(t, "/")
	tree := flightTree(t, body)
	arr, ok := tree.([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("expected array root, got %T: %v", tree, tree)
	}
	if arr[0] != "$" {
		t.Errorf("expected '$' as first element, got %v", arr[0])
	}
}

func TestFlightTree_HomePage_HasBranding(t *testing.T) {
	body := rsc(t, "/")
	// Async components land in later Flight rows, not row 0 — use raw body check.
	for _, want := range []string{"Dev", "Blog", "Read posts"} {
		if !flightContains(body, want) {
			t.Errorf("HomePage Flight missing branding text %q", want)
		}
	}
}

// ─── Route group layout presence ──────────────────────────────────────────────

func TestRSC_BlogLayout_WrapsPostsPage(t *testing.T) {
	body := rsc(t, "/posts")
	// (blog)/layout.tsx contributes the sidebar; (blog)/posts/layout.tsx adds the section badge
	if !flightContains(body, "Topics") {
		t.Error("Posts page missing (blog) sidebar layout content 'Topics'")
	}
	if !flightContains(body, "Blog") {
		t.Error("Posts page missing (blog)/posts section badge 'Blog'")
	}
}

func TestRSC_RevisionsPage(t *testing.T) {
	body := rsc(t, "/posts/go-react-ssr/revisions")
	for _, want := range []string{"v3", "v2", "v1", "Published", "Initial draft"} {
		if !flightContains(body, want) {
			t.Errorf("RevisionsPage missing %q", want)
		}
	}
}

func TestRSC_RevisionDetailPage(t *testing.T) {
	body := rsc(t, "/posts/go-react-ssr/revisions/v2")
	for _, want := range []string{"v2", "Building SSR with Go and React", "esbuild two-pass"} {
		if !flightContains(body, want) {
			t.Errorf("RevisionDetail missing %q", want)
		}
	}
}

func TestRSC_RevisionDetailPage_NotFound(t *testing.T) {
	app := requireApp(t)
	req := httptest.NewRequest("GET", "/posts/go-react-ssr/revisions/v99", nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.HandleRoute(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	if !strings.Contains(string(body), ":E{") && !strings.Contains(string(body), ":E\"") {
		t.Errorf("expected Flight error row for missing revision, got: %s", string(body)[:min(len(string(body)), 200)])
	}
}

func TestRSC_DocsIndex(t *testing.T) {
	body := rsc(t, "/docs")
	for _, want := range []string{"Documentation", "Getting Started", "Core Concepts"} {
		if !flightContains(body, want) {
			t.Errorf("DocsIndex missing %q", want)
		}
	}
}

func TestRSC_DocsSection(t *testing.T) {
	body := rsc(t, "/docs/getting-started")
	if !flightContains(body, "getting-started") && !flightContains(body, "Getting Started") {
		t.Error("DocsSection missing section name")
	}
}

func TestRSC_DocsNestedPage(t *testing.T) {
	body := rsc(t, "/docs/getting-started/installation")
	if !flightContains(body, "installation") {
		t.Error("DocsNestedPage missing 'installation'")
	}
}

func TestRSC_AllRoutesRender(t *testing.T) {
	paths := []string{"/", "/posts", "/posts/go-react-ssr", "/projects", "/projects/1", "/about", "/profile",
		"/posts/go-react-ssr/revisions", "/posts/go-react-ssr/revisions/v1",
		"/docs", "/docs/getting-started", "/docs/getting-started/installation"}
	for _, p := range paths {
		body := rsc(t, p)
		if !strings.Contains(body, "0:") {
			t.Errorf("page %s: missing Flight 0: row", p)
		}
	}
}

// ─── Client bundle tests ───────────────────────────────────────────────────────

func TestClientBundle_Served(t *testing.T) {
	app := requireApp(t)
	if app.ClientEntryScript == "" {
		t.Fatal("ClientEntryScript is empty")
	}
	if !strings.HasPrefix(app.ClientEntryScript, "/public/") {
		t.Errorf("ClientEntryScript should start with /public/, got %q", app.ClientEntryScript)
	}
}

func TestClientBundle_FileExists(t *testing.T) {
	app := requireApp(t)
	rel := strings.TrimPrefix(app.ClientEntryScript, "/public/")
	path := "../public/" + rel
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("client bundle not found at %q: %v", path, err)
	}
	if info.Size() < 10_000 {
		t.Errorf("client bundle suspiciously small (%d bytes)", info.Size())
	}
}

func TestClientBundle_NoWebpackRequire(t *testing.T) {
	app := requireApp(t)
	rel := strings.TrimPrefix(app.ClientEntryScript, "/public/")
	data, err := os.ReadFile("../public/" + rel)
	if err != nil {
		t.Fatalf("read client bundle: %v", err)
	}
	if strings.Contains(string(data), "__webpack_require__") {
		t.Error("client bundle contains __webpack_require__")
	}
}

// ─── Concurrent requests ───────────────────────────────────────────────────────

func TestRSC_Concurrent(t *testing.T) {
	app := requireApp(t)
	const n = 10
	results := make(chan error, n)
	for range n {
		go func() {
			req := httptest.NewRequest("GET", "/posts", nil)
			req.Header.Set("Content-Type", "text/x-component")
			w := httptest.NewRecorder()
			app.HandleRoute(w, req)
			body, _ := io.ReadAll(w.Result().Body)
			if !strings.Contains(string(body), "0:") {
				results <- fmt.Errorf("bad output: %s", string(body)[:min(len(string(body)), 100)])
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

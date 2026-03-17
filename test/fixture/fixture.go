// Package fixture provides the AppFixture interface and VM-specific
// implementations for the e2e test suite.
//
// Each VM×renderer×bundler combination registers itself via its file's init().
// Tests call ForEachApp or ForEachReactApp to run assertions against every
// registered fixture in a table-driven sub-test.
//
// To add a new VM: create <vmname>_fixture.go, implement AppFixture, and call
// Register() from init(). The test files require no changes.
package fixture

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gojsx/framework"
	"gojsx/framework/contract"
)

// PolyfillFixture is a polyfill-enabled JS execution context for one VM.
type PolyfillFixture interface {
	// Enable installs all polyfills into the context.
	Enable() error
	// Eval executes src and returns an error if JS throws.
	Eval(src string) error
}

// AppFixture is a lazily-built, fully-wired application for one VM×renderer×bundler combination.
type AppFixture interface {
	// Name returns a composite identifier, e.g. "goja:react:esbuild".
	Name() string
	// VMName returns the JS engine name, e.g. "goja".
	VMName() string
	// Renderer returns the render engine name, e.g. "react".
	Renderer() string
	// Bundler returns the bundler name, e.g. "esbuild".
	Bundler() string
	// GetApp returns the built *framework.App, constructing it lazily on first call.
	GetApp(t *testing.T) *framework.App
	// NewPolyfill returns a fresh, polyfill-enabled JS context for this VM.
	// The context lifetime is scoped to t.
	NewPolyfill(t *testing.T) PolyfillFixture
}

// AppDir is the path to the test application, relative to the e2e package root.
const AppDir = "../../ui/apps/blog-e2e"

var fixtureList []AppFixture

// Register is called by each driver file's init().
func Register(f AppFixture) {
	fixtureList = append(fixtureList, f)
}

// Fixtures returns all registered AppFixtures.
func Fixtures() []AppFixture { return fixtureList }

// ForEachApp runs fn as a sub-test for every registered fixture.
func ForEachApp(t *testing.T, fn func(*testing.T, AppFixture)) {
	t.Helper()
	for _, f := range fixtureList {
		f := f
		t.Run(f.Name(), func(t *testing.T) {
			fn(t, f)
		})
	}
}

// ForEachReactApp is like ForEachApp but skips fixtures whose renderer is not "react".
func ForEachReactApp(t *testing.T, fn func(*testing.T, AppFixture)) {
	t.Helper()
	ForEachApp(t, func(t *testing.T, f AppFixture) {
		if f.Renderer() != "react" {
			t.Skipf("skipping: %s renderer is not react", f.Renderer())
		}
		fn(t, f)
	})
}

// ForEachVM runs fn in a sub-test for every registered VM, providing a fresh
// polyfill-enabled PolyfillFixture. The fixture lifetime is scoped to the sub-test.
func ForEachVM(t *testing.T, fn func(t *testing.T, f PolyfillFixture)) {
	t.Helper()
	for _, fix := range fixtureList {
		fix := fix
		t.Run(fix.VMName(), func(t *testing.T) {
			f := fix.NewPolyfill(t)
			if err := f.Enable(); err != nil {
				t.Fatal(err)
			}
			fn(t, f)
		})
	}
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// RSC makes a React Server Component (text/x-component) request and returns the body.
// Fails the test if the response is not HTTP 200.
func RSC(t *testing.T, f AppFixture, path string) string {
	t.Helper()
	app := f.GetApp(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", path, nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("RSC GET %s → %d", path, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// RSCAny makes an RSC request and returns the status code + body without failing on non-200.
func RSCAny(t *testing.T, f AppFixture, path string) (int, string) {
	t.Helper()
	app := f.GetApp(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", path, nil)
	req.Header.Set("Content-Type", "text/x-component")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	return w.Result().StatusCode, string(body)
}

// Page makes a normal HTML request and returns the body. Fails on non-200.
func Page(t *testing.T, f AppFixture, path string) string {
	t.Helper()
	app := f.GetApp(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", path, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s → %d", path, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// PageAny makes an HTML request and returns the status + body without failing on non-200.
func PageAny(t *testing.T, f AppFixture, path string) (int, string) {
	t.Helper()
	app := f.GetApp(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", path, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	body, _ := io.ReadAll(w.Result().Body)
	return w.Result().StatusCode, string(body)
}

// FlightTree parses the root Flight row (0:...) as JSON and returns it.
func FlightTree(t *testing.T, body string) any {
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

// FlightContains reports whether substr appears anywhere in the Flight body.
func FlightContains(body, substr string) bool { return strings.Contains(body, substr) }

// ── Shared bridge config ──────────────────────────────────────────────────────

// SharedBridge returns the test data bridge used by all e2e fixtures.
func SharedBridge() contract.BridgeConfig {
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

	return contract.BridgeConfig{
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
}

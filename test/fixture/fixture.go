// Package fixture provides the test fixture interfaces and registration helpers
// for the Pola e2e and polyfill test suites.
//
// There are two independent registries:
//   - AppFixture implementations: registered with Register (one file per engine+combo)
//   - Polyfill VM fixtures: registered with RegisterPolyfillVM (one file per engine)
//
// ForEachApp automatically iterates all registered AppFixtures.
// ForEachVM iterates all registered polyfill VM fixtures.
package fixture

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/globals"
)

// ── Interfaces ────────────────────────────────────────────────────────────────

// PolyfillFixture is a polyfill-enabled JS execution context for one VM.
type PolyfillFixture interface {
	// Enable installs all polyfills into the context.
	Enable() error
	// Eval executes src and returns an error if JS throws.
	Eval(src string) error
}

// AppFixture is a fully-wired application fixture for one engine+renderer+bundler combination.
type AppFixture interface {
	// Name returns a composite identifier, e.g. "goja:react:esbuild".
	Name() string
	// EngineName returns the JS engine name, e.g. "goja".
	EngineName() string
	// RendererName returns the renderer name, e.g. "react".
	RendererName() string
	// BundlerName returns the bundler name, e.g. "esbuild".
	BundlerName() string
	// GetApp returns the built *core.App, constructing it lazily on first call.
	GetApp(t *testing.T) *core.App
	// NewPolyfill returns a fresh, polyfill-enabled JS context for this engine.
	NewPolyfill(t *testing.T) PolyfillFixture
}

// ── Registries ────────────────────────────────────────────────────────────────

var (
	fixtureList []AppFixture
	polyfillVMs []polyfillVMEntry
	fixtureMu   sync.Mutex
)

type polyfillVMEntry struct {
	name    string
	factory func(t *testing.T) PolyfillFixture
}

// Register is called by combo files' init() to register an AppFixture.
func Register(f AppFixture) {
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	fixtureList = append(fixtureList, f)
}

// RegisterPolyfillVM registers a polyfill-only VM fixture by name.
// Called from init() in test/vm/ files.
func RegisterPolyfillVM(name string, factory func(t *testing.T) PolyfillFixture) {
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	polyfillVMs = append(polyfillVMs, polyfillVMEntry{name: name, factory: factory})
}

// ── Public iteration helpers ──────────────────────────────────────────────────

// AppDir is the path to the test application, relative to the e2e package root.
const AppDir = "../../ui/apps/blog-e2e-react"

// ForEachApp runs fn as a sub-test for every registered AppFixture.
func ForEachApp(t *testing.T, fn func(*testing.T, AppFixture)) {
	t.Helper()
	fixtureMu.Lock()
	list := make([]AppFixture, len(fixtureList))
	copy(list, fixtureList)
	fixtureMu.Unlock()

	for _, f := range list {
		f := f
		t.Run(f.Name(), func(t *testing.T) { fn(t, f) })
	}
}

// ForEachReactApp is like ForEachApp but skips non-React renderer fixtures.
func ForEachReactApp(t *testing.T, fn func(*testing.T, AppFixture)) {
	t.Helper()
	ForEachApp(t, func(t *testing.T, f AppFixture) {
		if f.RendererName() != "react" {
			t.Skipf("skipping: %s renderer is not react", f.RendererName())
		}
		fn(t, f)
	})
}

// ForEachVM runs fn in a sub-test for each registered polyfill VM fixture,
// providing a fresh polyfill-enabled PolyfillFixture.
func ForEachVM(t *testing.T, fn func(t *testing.T, f PolyfillFixture)) {
	t.Helper()
	fixtureMu.Lock()
	vms := make([]polyfillVMEntry, len(polyfillVMs))
	copy(vms, polyfillVMs)
	fixtureMu.Unlock()

	seen := map[string]bool{}
	for _, vm := range vms {
		if seen[vm.name] {
			continue
		}
		seen[vm.name] = true
		vm := vm
		t.Run(vm.name, func(t *testing.T) {
			pf := vm.factory(t)
			if err := pf.Enable(); err != nil {
				t.Fatal(err)
			}
			fn(t, pf)
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

// ── Shared test injector ──────────────────────────────────────────────────────

// SharedInjector returns a RuntimeInjector pre-loaded with the blog-e2e test data.
// All e2e fixtures should include this in their Registry.Injectors.
func SharedInjector() core.RuntimeInjector {
	return &testInjector{fns: sharedFunctions()}
}

// testInjector is a minimal RuntimeInjector for tests that doesn't depend on samber/do.
type testInjector struct {
	fns map[string]func(args []any) (any, error)
}

func (i *testInjector) Name() string { return "test-injector" }
func (i *testInjector) Capabilities() []core.InjectionCapability {
	caps := make([]core.InjectionCapability, 0, len(i.fns))
	for name := range i.fns {
		caps = append(caps, core.InjectionCapability{Name: name})
	}
	return caps
}
// asyncDIRuntime is the optional interface for runtimes that support
// async (Promise + goroutine) dependency injection.
type asyncDIRuntime interface {
	SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error
}

func (i *testInjector) Inject(_ context.Context, runtime core.JSRuntime) error {
	if ar, ok := runtime.(asyncDIRuntime); ok {
		return ar.SetDependencyInjection(i.fns)
	}
	fns := make(map[string]any, len(i.fns))
	for name, fn := range i.fns {
		fns[name] = fn
	}
	return runtime.Set(globals.BridgeObject, fns)
}

// sharedFunctions returns the blog-e2e test data as injector functions.
func sharedFunctions() map[string]func(args []any) (any, error) {
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

	return map[string]func(args []any) (any, error){
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
	}
}

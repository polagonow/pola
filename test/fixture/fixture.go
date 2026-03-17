// Package fixture provides the test fixture interfaces and cross-product
// registration for e2e and polyfill test suites.
//
// There are two independent registries:
//   - VM implementations: registered with RegisterVM (one file per engine)
//   - Bundler+renderer combos: registered with RegisterBundlerRenderer (one file per combo)
//
// ForEachApp automatically runs every VM × every bundler+renderer combo.
// To add a new VM: create a file in test/vm/, implement VMFixture, call RegisterVM.
// To add a new bundler+renderer pair: create a file in test/combo/, implement
// BundlerRendererFixture, call RegisterBundlerRenderer. No other files need changing.
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

	"gojsx/framework"
	"gojsx/framework/contract"
)

// ── Interfaces ────────────────────────────────────────────────────────────────

// PolyfillFixture is a polyfill-enabled JS execution context for one VM.
type PolyfillFixture interface {
	// Enable installs all polyfills into the context.
	Enable() error
	// Eval executes src and returns an error if JS throws.
	Eval(src string) error
}

// VMFixture provides the VM-specific half of an e2e fixture.
// Register implementations with RegisterVM from an init() function.
type VMFixture interface {
	// VMName returns the JS engine identifier, e.g. "goja".
	VMName() string
	// VMFactory returns the constructor used to create a VMFactory from a bundle.
	VMFactory() func([]byte) (framework.VMFactory, error)
	// NewPolyfill returns a fresh polyfill-enabled context scoped to t.
	NewPolyfill(t *testing.T) PolyfillFixture
}

// BundlerRendererFixture provides one bundler+renderer combination.
// Register combinations with RegisterBundlerRenderer from an init() function.
type BundlerRendererFixture interface {
	// BundlerName returns the bundler identifier, e.g. "esbuild".
	BundlerName() string
	// RendererName returns the renderer identifier, e.g. "react".
	RendererName() string
	// NewBundler returns a fresh Bundler instance.
	NewBundler() framework.Bundler
	// NewRendererFactory returns the RendererFactory used by framework.Config.
	NewRendererFactory() func(framework.VMPool, framework.StreamProtocol, contract.BridgeConfig) framework.Renderer
}

// AppFixture is a lazily-built, fully-wired application for one VM×renderer×bundler combination.
// Implementations are generated automatically from the VM × combo cross-product.
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
	NewPolyfill(t *testing.T) PolyfillFixture
}

// ── Registries ────────────────────────────────────────────────────────────────

var (
	vmList    []VMFixture
	comboList []BundlerRendererFixture

	compositeMu    sync.Mutex
	compositeCache = map[string]*compositeFixture{}
)

// RegisterVM is called by VM driver files' init().
func RegisterVM(v VMFixture) { vmList = append(vmList, v) }

// RegisterBundlerRenderer is called by bundler+renderer combo files' init().
func RegisterBundlerRenderer(c BundlerRendererFixture) { comboList = append(comboList, c) }

// ── compositeFixture ──────────────────────────────────────────────────────────

// compositeFixture is a lazily-built AppFixture for one VM × bundler+renderer pair.
type compositeFixture struct {
	vm    VMFixture
	combo BundlerRendererFixture
	once  sync.Once
	app   *framework.App
	err   error
}

func (f *compositeFixture) Name() string {
	return f.vm.VMName() + ":" + f.combo.RendererName() + ":" + f.combo.BundlerName()
}
func (f *compositeFixture) VMName() string   { return f.vm.VMName() }
func (f *compositeFixture) Renderer() string { return f.combo.RendererName() }
func (f *compositeFixture) Bundler() string  { return f.combo.BundlerName() }

func (f *compositeFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:          AppDir,
			GlobalBridge:    SharedBridge(),
			NewVM:           f.vm.VMFactory(),
			Bundler:         f.combo.NewBundler(),
			RendererFactory: f.combo.NewRendererFactory(),
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

func (f *compositeFixture) NewPolyfill(t *testing.T) PolyfillFixture { return f.vm.NewPolyfill(t) }

// allFixtures returns the VM × combo cross-product, caching composites so the
// same instance (and its sync.Once) is reused across test functions.
func allFixtures() []AppFixture {
	compositeMu.Lock()
	defer compositeMu.Unlock()

	var result []AppFixture
	for _, vm := range vmList {
		for _, combo := range comboList {
			key := vm.VMName() + ":" + combo.RendererName() + ":" + combo.BundlerName()
			if _, ok := compositeCache[key]; !ok {
				compositeCache[key] = &compositeFixture{vm: vm, combo: combo}
			}
			result = append(result, compositeCache[key])
		}
	}
	return result
}

// ── Public iteration helpers ──────────────────────────────────────────────────

// AppDir is the path to the test application, relative to the e2e package root.
const AppDir = "../../ui/apps/blog-e2e"

// ForEachApp runs fn as a sub-test for every registered VM × bundler+renderer fixture.
func ForEachApp(t *testing.T, fn func(*testing.T, AppFixture)) {
	t.Helper()
	for _, f := range allFixtures() {
		f := f
		t.Run(f.Name(), func(t *testing.T) { fn(t, f) })
	}
}

// ForEachReactApp is like ForEachApp but skips non-React renderer fixtures.
func ForEachReactApp(t *testing.T, fn func(*testing.T, AppFixture)) {
	t.Helper()
	ForEachApp(t, func(t *testing.T, f AppFixture) {
		if f.Renderer() != "react" {
			t.Skipf("skipping: %s renderer is not react", f.Renderer())
		}
		fn(t, f)
	})
}

// ForEachVM runs fn in a sub-test for each registered VM (one sub-test per VM,
// deduplicated across combos), providing a fresh polyfill-enabled PolyfillFixture.
func ForEachVM(t *testing.T, fn func(t *testing.T, f PolyfillFixture)) {
	t.Helper()
	seen := map[string]bool{}
	for _, f := range allFixtures() {
		if seen[f.VMName()] {
			continue
		}
		seen[f.VMName()] = true
		f := f
		t.Run(f.VMName(), func(t *testing.T) {
			pf := f.NewPolyfill(t)
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

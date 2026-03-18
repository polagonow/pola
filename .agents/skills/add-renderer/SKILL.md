---
name: add-renderer
description: Add a new renderer to the Pola framework. Use when asked to add, integrate, or wire up a new UI renderer such as Vue, Svelte, Solid, or any alternative to React for server-side rendering.
---

A renderer implements `core.Renderer`. It declares which file extensions it handles
(used by the router to scan for routes) and renders pages given a `RenderRequest`.
Full renderers also implement `core.BundleLoader` to receive the compiled server bundle.

## Files to create

| File | Purpose |
|------|---------|
| `renderer/<name>/<name>.go` | `core.Renderer` implementation |
| `renderer/<name>/register.go` | `init()` that calls `core.RegisterRenderer` — gated by build tag |
| `test/combo/esbuild_<name>.go` | Test combo: esbuild × new renderer × Goja engine |

---

## Step 1 — Implement `core.Renderer`

**`renderer/<name>/<name>.go`** — reference: `renderer/react/react.go`

```go
//go:build <name>

package myrenderer

import (
    "context"
    "fmt"

    "github.com/polagonow/pola/core"
)

// Renderer implements core.Renderer for <name>.
type Renderer struct{}

// New returns a new Renderer.
func New() *Renderer { return &Renderer{} }

// Name implements core.Renderer.
func (r *Renderer) Name() string { return "<name>" }

// FileExtensions declares which file extensions this renderer handles.
// The router uses these to scan for page files in the app directory.
func (r *Renderer) FileExtensions() []string {
    return []string{".<ext>"} // e.g. ".vue", ".svelte", ".templ"
}

// Capabilities lists optional features this renderer supports.
func (r *Renderer) Capabilities() []core.Capability { return nil }

// Render handles a single page request.
func (r *Renderer) Render(ctx context.Context, req core.RenderRequest) (core.RenderResult, error) {
    // req fields:
    //   Route        core.Route    — matched route
    //   Params       map[string]any — dynamic path params
    //   HTTPRequest  *http.Request
    //   IsRSC        bool          — true if client requests text/x-component (Flight)
    //   BundleOutput *core.BundleOutput

    return core.RenderResult{}, fmt.Errorf("<name>: Render not yet implemented")
}
```

### RenderResult fields

| Field | Type | Description |
|-------|------|-------------|
| `ContentType` | `string` | HTTP Content-Type (e.g. `"text/html"`, `"text/x-component"`) |
| `Body` | `[]byte` | Full response body (use for non-streaming renderers) |
| `Stream` | `io.Reader` | Streaming response body (preferred for RSC/Suspense) |
| `StatusCode` | `int` | HTTP status (0 = 200 OK) |

### Full renderer with JS bundle (SSR)

If the renderer executes JS (like React), implement `core.BundleLoader` to receive
the compiled server bundle:

```go
// LoadBundle implements core.BundleLoader.
// Called by the pipeline after bundling completes.
func (r *Renderer) LoadBundle(engine core.JSEngine, bundle []byte) error {
    factory, ok := engine.(core.SSRPoolFactory)
    if !ok {
        return fmt.Errorf("<name>: engine %s does not support SSR pool", engine.Name())
    }
    pool, err := factory.NewSSRPool(bundle)
    if err != nil {
        return err
    }
    r.pool = pool
    return nil
}
```

## Step 2 — Register via init()

**`renderer/<name>/register.go`**

```go
//go:build <name>

package myrenderer

import "github.com/polagonow/pola/core"

func init() {
    core.RegisterRenderer(func() core.Renderer { return New() })
}
```

## Step 3 — Add a test combo

**`test/combo/esbuild_<name>.go`** — reference: `test/combo/esbuild_react.go`

```go
//go:build goja && esbuild && <name> && nextjs

package combo

import (
    "sync"
    "testing"

    pola "github.com/polagonow/pola"
    "github.com/polagonow/pola/core"
    gojaengine "github.com/polagonow/pola/engine/goja"
    "github.com/polagonow/pola/test/fixture"

    _ "github.com/polagonow/pola/bundler/esbuild"
    _ "github.com/polagonow/pola/fs/osfs"
    _ "github.com/polagonow/pola/logger/slog"
    _ "github.com/polagonow/pola/renderer/<name>"
    _ "github.com/polagonow/pola/router/nextjs"
)

func init() { fixture.Register(&esbuildMyRendererGojaFixture{}) }

type esbuildMyRendererGojaFixture struct {
    once sync.Once
    app  *core.App
    err  error
}

func (f *esbuildMyRendererGojaFixture) Name() string         { return "goja:<name>:esbuild" }
func (f *esbuildMyRendererGojaFixture) EngineName() string   { return "goja" }
func (f *esbuildMyRendererGojaFixture) RendererName() string { return "<name>" }
func (f *esbuildMyRendererGojaFixture) BundlerName() string  { return "esbuild" }

func (f *esbuildMyRendererGojaFixture) GetApp(t *testing.T) *core.App {
    t.Helper()
    f.once.Do(func() {
        f.app, f.err = pola.New(&core.Config{
            WebAppPath: fixture.AppDir,
            Registry: &core.Registry{
                Engine:    gojaengine.NewEngine(),
                Injectors: []core.RuntimeInjector{fixture.SharedInjector()},
            },
        })
    })
    if f.err != nil {
        t.Fatalf("%s: build failed: %v", f.Name(), f.err)
    }
    return f.app
}

func (f *esbuildMyRendererGojaFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
    return nil // fill in if polyfill tests are needed
}
```

## Verify

```
go build -tags "goja esbuild <name> nextjs" ./...
go test -tags "goja esbuild <name> nextjs" -v -run TestHTMLShell ./test/e2e/...
go test -tags "goja esbuild <name> nextjs" -v ./test/e2e/...
```

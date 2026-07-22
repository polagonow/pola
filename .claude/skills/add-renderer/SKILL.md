---
name: add-renderer
description: Add a new renderer to the Pola framework. Use when asked to add, integrate, or wire up a new UI renderer such as Vue, Svelte, Solid, or any alternative to React for server-side rendering.
---

A renderer implements `core.Renderer`. It declares which file extensions it handles
(used by the router to scan for routes) and serves matched page requests as an
`http.Handler`. Renderers register via a `Plugin()` function (DI) — no init()
registries, no build tags. Existing implementations: `renderer/react/`, `renderer/vue/`,
`renderer/svelte/`, `renderer/angular/`, `renderer/nativersc/`, `renderer/templ/`,
`renderer/htmx/`, `renderer/mdx/`.

## Files to create

| File | Purpose |
|------|---------|
| `renderer/<name>/<name>.go` | `core.Renderer` implementation |
| `renderer/<name>/plugin.go` | `Plugin() core.Plugin` that provides the renderer into the registry |
| `test/combo/esbuild_<name>.go` | Test combo: esbuild × new renderer × Goja engine |

---

## Step 1 — Implement `core.Renderer`

**`renderer/<name>/<name>.go`** — reference: `renderer/react/react.go`

```go
package myrenderer

import (
    "net/http"

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

// ServeHTTP renders the matched page request. The pipeline routes matched
// page requests here; use the request context / injected RenderDeps to find
// the matched route, params, and whether this is an RSC (text/x-component)
// request.
func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    http.Error(w, "<name>: not yet implemented", http.StatusNotImplemented)
}
```

### Optional interfaces the pipeline checks for

| Interface | Method | When to implement |
|-----------|--------|-------------------|
| `core.RenderDepsAware` | `SetRenderDeps(core.RenderDeps)` | You need framework deps (shell, cache, logger, router, …); called once after wiring |
| `core.BundleLoader` | `LoadBundle(engine core.JSEngine, bundle []byte) error` | The renderer executes JS — receives the compiled server bundle after every (re)build |
| `core.ServerActionInvoker` | — | The renderer can execute RSC `'use server'` actions (serves `/_pola/action`) |

A JS-executing renderer typically builds its VM pool inside `LoadBundle`:

```go
// LoadBundle implements core.BundleLoader.
func (r *Renderer) LoadBundle(engine core.JSEngine, bundle []byte) error {
    factory, ok := engine.(core.SSRPoolFactory)
    if !ok {
        return fmt.Errorf("<name>: engine %s does not support SSR pools", engine.Name())
    }
    pool, err := factory.NewSSRPool(bundle)
    if err != nil {
        return err
    }
    r.pool = pool
    return nil
}
```

### Bundler-specific build plugins

If the renderer needs bundler-native build plugins (workspace resolution, client
boundary probing, module stubs), provide a `core.BundlePluginProvider` from a
bundler-specific subpackage, and expose a composite plugin — mirror
`renderer/react/esbuild/plugin.go`, which registers `react.Plugin()` plus an
esbuild `BundlePluginProvider`.

## Step 2 — Register via `Plugin()`

**`renderer/<name>/plugin.go`** — reference: `renderer/react/plugin.go`

```go
package myrenderer

import (
    "github.com/polagonow/pola/core"
    "github.com/polagonow/pola/shell"
)

// Plugin returns the <name> renderer plugin.
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.Renderer](r, New())
            // If the renderer mounts into an HTML shell, provide one too:
            core.ProvideValue[core.HTMLShell](r, shell.New(`<div id="root"></div>`))
        },
    }
}
```

Apps activate it with `builder.Use(myrenderer.Plugin())`; the CLI selects it from
`Polafile.hcl`'s `renderer = "<name>"` / the `--renderer` flag.

## Step 3 — Add a test combo

**`test/combo/esbuild_<name>.go`** — reference: `test/combo/esbuild_react.go`.
Register an `AppFixture` whose plugin list swaps in your renderer:

```go
package combo

import (
    "context"
    "sync"
    "testing"

    "github.com/polagonow/pola"
    "github.com/polagonow/pola/core"
    "github.com/polagonow/pola/test/fixture"

    myrenderer "github.com/polagonow/pola/renderer/<name>"
    // ...same supporting plugins as esbuild_react.go (goja, esbuild, css, router,
    // fs, logger, middleware, cache, metrics, tracing).
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
        builder := pola.NewApp(core.WithWebAppDir(fixture.AppDir))
        builder.Use(myrenderer.Plugin() /* + the rest of the plugin list */)
        f.app, f.err = pola.BuildApp(context.Background(), builder)
    })
    if f.err != nil {
        t.Fatalf("%s: build failed: %v", f.Name(), f.err)
    }
    return f.app
}

func (f *esbuildMyRendererGojaFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
    return nil // fill in if polyfill tests should run for this combo
}
```

Renderer-agnostic suites (`fixture.ForEachApp`) will pick the combo up automatically;
React-specific suites use `fixture.ForEachReactApp` and skip it.

## Verify

```
go build ./...
go test -v -run TestHTMLShell ./test/e2e/...
go test -v ./test/e2e/...
```

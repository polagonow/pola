---
name: add-bundler
description: Add a new bundler to the Pola framework. Use when asked to add, integrate, or wire up a new JS bundler such as Vite, Rollup, Parcel, Webpack, or any other build tool.
---

A bundler implements `core.Bundler`. It takes source files from the app directory
and produces a server-side JS bundle (run inside the VM) plus client-side assets
(served to the browser). The framework calls `Build` at startup and `Watch` in dev
mode. Bundlers register via a `Plugin()` function (DI) — no init() registries, no
build tags on implementation files. Existing implementations: `bundler/esbuild/`,
`bundler/rollup/`, `bundler/vite/`.

## Files to create

| File | Purpose |
|------|---------|
| `bundler/<name>/<name>.go` | `core.Bundler` implementation |
| `bundler/<name>/plugin.go` | `Plugin() core.Plugin` that provides the bundler into the registry |
| `test/combo/<name>_react.go` | Test combo: new bundler × React renderer × Goja engine |

---

## Step 1 — Implement `core.Bundler`

**`bundler/<name>/<name>.go`** — reference: `bundler/esbuild/esbuild.go`

```go
package mybundler

import (
    "context"
    "fmt"

    "github.com/polagonow/pola/core"
)

// Bundler implements core.Bundler using <name>.
type Bundler struct{}

// New returns a new Bundler.
func New() *Bundler { return &Bundler{} }

// Name implements core.Bundler.
func (b *Bundler) Name() string { return "<name>" }

// Build runs the two-pass bundling pipeline.
// Pass 1: client bundle (browser ESM) + manifest.
// Pass 2: server bundle (CJS, react-server conditions) with client proxy stubs.
func (b *Bundler) Build(_ context.Context, req core.BundleInput) (*core.BundleOutput, error) {
    // req fields (core/types.go):
    //   AppDir                 string            — root of the user's app
    //   OutDir                 string            — write client assets here
    //   AssetsURLPath          string            — URL prefix for browser assets
    //   ClientEntry            string            — path to the browser bootstrap entry (_client.tsx)
    //   ClientComponents       []string          — abs paths of "use client" files
    //   Dev                    bool              — enable source maps / fast builds
    //   ServerEntry            string            — output path for the server bundle
    //   ServerEntryContent     string            — pre-generated server entry TS source
    //   ServerBundleConditions []string          — package.json export conditions (e.g. ["react-server"])
    //   ServerBundleDefines    map[string]string — renderer-provided defines for the server pass
    //   External               []string          — packages to mark external
    //   PublicEnvVars          map[string]string — POLA_PUBLIC_* vars injected as process.env.*
    //   CSSProcessor           core.CSS          — optional CSS processor for the client pass

    return &core.BundleOutput{
        ServerBundle:   []byte("/* compiled server JS */"),
        ClientFiles:    map[string][]byte{},
        ClientEntryURL: req.AssetsURLPath + "/main.js",
        ManifestJSON:   []byte("{}"),
        ImportURLs:     map[string]string{},
    }, nil
}

// Watch implements core.Bundler — emits a new BundleOutput on every rebuild.
// Return a channel that delivers rebuilt outputs until ctx is cancelled.
func (b *Bundler) Watch(_ context.Context, _ core.BundleInput) (<-chan *core.BundleOutput, error) {
    return nil, fmt.Errorf("<name>: Watch not yet implemented")
}
```

### BundleOutput fields

| Field | Type | Description |
|-------|------|-------------|
| `ServerBundle` | `[]byte` | CJS bundle executed inside the JS engine |
| `ClientFiles` | `map[string][]byte` | Browser assets: relative path → content |
| `ClientEntryURL` | `string` | Browser URL of the client entry script |
| `ManifestJSON` | `[]byte` | Webpack-format client component manifest |
| `ImportURLs` | `map[string]string` | Component module ID → browser chunk URL |

### Renderer-native build plugins

Renderers can contribute bundler-native plugins through `core.BundlePluginProvider`
(`ClientPlugins` / `ServerPlugins` / `ProbePlugins` / `ClientModuleStub`). The React
renderer ships an esbuild-specific provider in `renderer/react/esbuild/`. If your
bundler should support React RSC, add an equivalent integration package
(`renderer/react/<name>/`) that type-asserts those `[]any` plugins to your bundler's
native plugin type — see `renderer/react/esbuild/plugins.go`.

## Step 2 — Register via `Plugin()`

**`bundler/<name>/plugin.go`** — reference: `bundler/esbuild/plugin.go`

```go
package mybundler

import "github.com/polagonow/pola/core"

// Plugin returns the <name> bundler plugin.
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.Bundler](r, New())
        },
    }
}
```

Apps activate it with `builder.Use(mybundler.Plugin())`; the CLI selects it from
`Polafile.hcl`'s `bundler = "<name>"` / the `--bundler` flag.

## Step 3 — Add a test combo

**`test/combo/<name>_react.go`** — reference: `test/combo/esbuild_react.go`.
A combo registers a fully-wired app fixture: a plugin list + a lazily-built
`*core.App`.

```go
package combo

import (
    "context"
    "sync"
    "testing"

    "github.com/polagonow/pola"
    "github.com/polagonow/pola/core"
    "github.com/polagonow/pola/engine/goja"
    "github.com/polagonow/pola/test/fixture"

    mybundler "github.com/polagonow/pola/bundler/<name>"
    // ...same supporting plugins as esbuild_react.go: css, router, fs, logger,
    // logging/recovery middleware, cache, metrics, tracing, react renderer glue.
)

func init() { fixture.Register(&myBundlerReactGojaFixture{}) }

type myBundlerReactGojaFixture struct {
    once sync.Once
    app  *core.App
    err  error
}

func (f *myBundlerReactGojaFixture) Name() string         { return "goja:react:<name>" }
func (f *myBundlerReactGojaFixture) EngineName() string   { return "goja" }
func (f *myBundlerReactGojaFixture) RendererName() string { return "react" }
func (f *myBundlerReactGojaFixture) BundlerName() string  { return "<name>" }

func (f *myBundlerReactGojaFixture) GetApp(t *testing.T) *core.App {
    t.Helper()
    f.once.Do(func() {
        builder := pola.NewApp(core.WithWebAppDir(fixture.AppDir))
        builder.Use(goja.Plugin(), mybundler.Plugin() /* + the rest of the plugin list */)
        f.app, f.err = pola.BuildApp(context.Background(), builder)
    })
    if f.err != nil {
        t.Fatalf("%s: build failed: %v", f.Name(), f.err)
    }
    return f.app
}

func (f *myBundlerReactGojaFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
    // return a goja polyfill fixture (copy gojaPolyfillFixture from esbuild_react.go)
    return nil
}
```

No change to `test/e2e/rendering_test.go` — it blank-imports `test/combo` and every
registered fixture runs automatically via `fixture.ForEachApp`.

## Verify

```
go build ./...
go test -v -run TestHTMLShell ./test/e2e/...
go test -v ./test/e2e/...
```

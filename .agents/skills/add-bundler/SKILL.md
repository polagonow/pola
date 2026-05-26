---
name: add-bundler
description: Add a new bundler to the Pola framework. Use when asked to add, integrate, or wire up a new JS bundler such as Vite, Rollup, Parcel, Webpack, or any other build tool.
---

A bundler implements `core.Bundler`. It takes source files from the app directory
and produces a server-side JS bundle (run inside the VM) plus client-side assets
(served to the browser). The framework calls `Build` at startup and `Watch` in dev mode.

## Files to create

| File | Purpose |
|------|---------|
| `bundler/<name>/<name>.go` | `core.Bundler` implementation |
| `bundler/<name>/register.go` | `init()` that calls `core.RegisterBundler` — gated by build tag |
| `test/combo/<name>_react.go` | Test combo: new bundler × React renderer × Goja engine |

---

## Step 1 — Implement `core.Bundler`

**`bundler/<name>/<name>.go`** — reference: `bundler/esbuild/esbuild.go`

```go
//go:build <name>

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
    // req fields:
    //   AppDir                 string            — root of the user's app
    //   OutDir                 string            — write client assets here
    //   AssetsURLPath          string            — URL prefix for browser assets
    //   ClientEntry            string            — browser bootstrap entry (e.g. "@pola/react/client")
    //   ClientComponents       []string          — "use client" files to expose to browser
    //   Dev                    bool              — enable source maps / fast builds
    //   ServerEntry            string            — path for the emitted server bundle file
    //   ServerEntryContent     string            — pre-generated server entry JS source
    //   ServerBundleConditions []string          — package.json export conditions (e.g. ["react-server"])
    //   ServerBundleDefines    map[string]string — additional esbuild defines for server pass
    //   External               []string          — packages to mark external
    //   PublicEnvVars          map[string]string — POLA_PUBLIC_* vars injected as process.env.*

    return &core.BundleOutput{
        ServerBundle:   []byte("/* compiled server JS */"),
        ClientFiles:    map[string][]byte{},
        ClientEntryURL: req.AssetsURLPath + "/main.js",
        ManifestJSON:   []byte("{}"),
        ImportURLs:     map[string]string{},
    }, nil
}

// Watch implements core.Bundler — rebuilds on file changes.
func (b *Bundler) Watch(_ context.Context, req core.BundleInput, onChange func(*core.BundleOutput)) error {
    return fmt.Errorf("<name>: Watch not yet implemented")
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

## Step 2 — Register via init()

**`bundler/<name>/register.go`**

```go
//go:build <name>

package mybundler

import "github.com/polagonow/pola/core"

func init() {
    core.RegisterBundler(func() core.Bundler { return New() })
}
```

## Step 3 — Add a test combo

**`test/combo/<name>_react.go`** — reference: `test/combo/esbuild_react.go`

```go
//go:build goja && <name> && react && nextjs

package combo

import (
    "sync"
    "testing"

    pola "github.com/polagonow/pola"
    "github.com/polagonow/pola/core"
    gojaengine "github.com/polagonow/pola/engine/goja"
    "github.com/polagonow/pola/test/fixture"

    _ "github.com/polagonow/pola/bundler/<name>"
    _ "github.com/polagonow/pola/fs/osfs"
    _ "github.com/polagonow/pola/logger/slog"
    _ "github.com/polagonow/pola/renderer/react"
    _ "github.com/polagonow/pola/router/nextjs"
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

func (f *myBundlerReactGojaFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
    // return a goja polyfill fixture (same as esbuild_react.go)
    return nil // fill in
}
```

No change to `test/e2e/ssr_rendering_test.go` — that file imports `test/combo` and all
registered fixtures run automatically.

## Verify

```
go build -tags "goja <name> react nextjs" ./...
go test -tags "goja <name> react nextjs" -v -run TestHTMLShell ./test/e2e/...
go test -tags "goja <name> react nextjs" -v ./test/e2e/...
```

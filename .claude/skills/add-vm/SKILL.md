---
name: add-vm
description: Add a new JavaScript engine (VM) to the Pola framework. Use when asked to add, integrate, or wire up a new JS runtime such as goja, v8go, sobek, quickjs, or any other engine.
---

A JS engine implements `core.JSEngine` and optionally `core.SSRPoolFactory` (for
React SSR). The engine declares which polyfills it needs; the pipeline injects only
those. Each runtime is a single JS execution context returned by `NewRuntime`.
Engines register via a `Plugin()` function (DI), not init()/build tags — the
implementation package has no build tags; only the test fixture file is tag-gated.

## Files to create

| File | Purpose |
|------|---------|
| `engine/<name>/<name>.go` | `JSEngine` (+ `SSRPoolFactory` if SSR-capable) |
| `engine/<name>/plugin.go` | `Plugin() core.Plugin` that provides the engine into the registry |
| `test/vm/<name>.go` | `PolyfillFixture` registration for the test suite — gated by `//go:build <name>` |

---

## Step 1 — Implement `core.JSEngine`

**`engine/<name>/<name>.go`** — reference: `engine/goja/goja.go`

```go
package myengine

import (
    "context"

    "github.com/polagonow/pola/core"
    "github.com/polagonow/pola/engine/polyfill"
)

// Engine implements core.JSEngine.
type Engine struct{}

func (e *Engine) Name() string { return "<name>" }

// RequiredPolyfills declares which Web API polyfills the engine needs.
// Engines with native support for an API should omit that polyfill.
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
    return []core.PolyfillID{
        polyfill.MicrotaskQueue,
        polyfill.TextEncoding,
        polyfill.MessageChannel,
        polyfill.ReadableStream,
        polyfill.AbortController,
        polyfill.WebpackRequire,
    }
}

// NewRuntime creates a single JS execution context.
func (e *Engine) NewRuntime(_ context.Context) (core.JSRuntime, error) {
    return &Runtime{}, nil
}

// Runtime implements core.JSRuntime.
type Runtime struct{}

func (r *Runtime) Eval(script string) (any, error)          { /* ... */ return nil, nil }
func (r *Runtime) Call(fn string, args ...any) (any, error) { /* ... */ return nil, nil }
func (r *Runtime) Set(name string, value any) error         { /* ... */ return nil }
func (r *Runtime) Dispose()                                 {}
```

### SSR-capable engines (React RSC support)

If the engine supports React Server Components, also implement `core.SSRPoolFactory`.
The pipeline calls `NewSSRPool(bundle)` after bundling; the pool hands out
`core.SSRRuntime` instances (a `JSRuntime` extended with streaming methods):

```go
// NewSSRPool implements core.SSRPoolFactory.
func (e *Engine) NewSSRPool(bundle []byte) (core.SSRPool, error) {
    // Compile the bundle once; return a pool with Acquire()/Release(rt).
    return &Pool{}, nil
}

// SSRRuntime = JSRuntime plus:
//   SetRequestContext(ctx map[string]any) error                       // injects __REQUEST__
//   CallRenderFunction(exportName, propsJSON string) (core.StreamHandle, error)
//   DrainStream(handle core.StreamHandle, w core.StreamWriter) (bool, error)
```

## Step 2 — Register via `Plugin()`

**`engine/<name>/plugin.go`** — reference: `engine/goja/plugin.go`

```go
package myengine

import "github.com/polagonow/pola/core"

// Plugin returns the <name> JS engine plugin.
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.JSEngine](r, &Engine{})
        },
    }
}
```

An app (or the CLI's generated wiring) activates it with `builder.Use(myengine.Plugin())`;
the CLI selects it from `Polafile.hcl`'s `engine = "<name>"` / the `--vm` flag.

## Step 3 — Register a polyfill fixture for tests

**`test/vm/<name>.go`** — reference: `test/vm/qjs.go`. This file **is** build-tag
gated so heavy engines (CGO, V8) only compile into test runs that ask for them.

```go
//go:build <name>

package vm

import (
    "testing"

    "github.com/polagonow/pola/engine/polyfill"
    "github.com/polagonow/pola/test/fixture"
)

func init() {
    fixture.RegisterPolyfillVM("<name>", func(t *testing.T) fixture.PolyfillFixture {
        rt := newBareRuntime() // create a raw engine context; t.Cleanup(...) to free it
        return &myPolyfillFixture{rt: rt}
    })
}

type myPolyfillFixture struct{ rt *bareRuntime }

// Enable installs every polyfill the engine declares in RequiredPolyfills.
func (f *myPolyfillFixture) Enable() error {
    reg := polyfill.DefaultRegistry()
    for _, src := range reg.Get(
        polyfill.MicrotaskQueue, polyfill.TextEncoding, polyfill.MessageChannel,
        polyfill.ReadableStream, polyfill.AbortController, polyfill.WebpackRequire,
    ) {
        if err := f.rt.Eval(src.Source); err != nil {
            return err
        }
    }
    return nil
}

func (f *myPolyfillFixture) Eval(src string) error { return f.rt.Eval(src) }
```

## Polyfills available (`engine/polyfill`)

| ID | What it installs |
|----|-----------------|
| `polyfill.MicrotaskQueue` | `queueMicrotask`, `__drainMicrotasks__` |
| `polyfill.TextEncoding` | `TextEncoder`, `TextDecoder` |
| `polyfill.MessageChannel` | `MessageChannel`, `MessagePort` |
| `polyfill.ReadableStream` | `ReadableStream`, `ReadableStreamDefaultController` |
| `polyfill.AbortController` | `AbortController`, `AbortSignal` |
| `polyfill.WebpackRequire` | `__webpack_require__` shim for RSC |
| `polyfill.Promise` | `Promise` (omit if engine has native Promises) |

Omit any polyfill your engine natively supports from `RequiredPolyfills()`.

## Verify

```
go build ./...
POLA_VM=<name> mage test          # or: go test -tags "<name> esbuild react nextjs tailwind" ./...
go test -tags "<name> esbuild react nextjs tailwind" -v ./engine/polyfill/...
```

The polyfill suite (`test/e2e/engine/polyfill_suite.go`) runs via `fixture.ForEachVM`
for every VM fixture whose build tag is active. To get full e2e coverage, also add a
combo fixture in `test/combo/` that includes your engine's `Plugin()` (see `add-bundler`
for the combo pattern).

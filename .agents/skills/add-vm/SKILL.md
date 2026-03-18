---
name: add-vm
description: Add a new JavaScript engine (VM) to the Pola framework. Use when asked to add, integrate, or wire up a new JS runtime such as goja, v8go, sobek, quickjs, or any other engine.
---

A JS engine implements `core.JSEngine` and optionally `core.SSRPoolFactory` (for
React SSR). The engine declares which polyfills it needs; the pipeline injects only
those. Each runtime is a single JS execution context returned by `NewRuntime`.

## Files to create

| File | Purpose |
|------|---------|
| `engine/<name>/<name>.go` | `JSEngine` (+ `SSRPoolFactory` if SSR-capable) |
| `engine/<name>/register.go` | `init()` that calls `core.RegisterEngine` — gated by build tag |
| `test/vm/<name>.go` | `PolyfillFixture` registration for the test suite |

---

## Step 1 — Implement `core.JSEngine`

**`engine/<name>/<name>.go`** — reference: `engine/goja/goja.go`

```go
//go:build <name>

package myengine

import (
    "context"

    "github.com/polagonow/pola/core"
    "github.com/polagonow/pola/engine/polyfill"
)

// Engine implements core.JSEngine.
type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

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

func (r *Runtime) Eval(script string) (any, error)         { /* ... */ return nil, nil }
func (r *Runtime) Call(fn string, args ...any) (any, error) { /* ... */ return nil, nil }
func (r *Runtime) Set(name string, value any) error        { /* ... */ return nil }
func (r *Runtime) Dispose()                                {}
```

### SSR-capable engines (React RSC support)

If the engine supports React Server Components, also implement `core.SSRPoolFactory`
and `core.SSRRuntime`. The pipeline calls `engine.NewSSRPool(bundle)` after bundling.

```go
// NewSSRPool implements core.SSRPoolFactory.
func (e *Engine) NewSSRPool(bundle []byte) (core.SSRPool, error) {
    // Compile bundle, create pool of SSRRuntime instances
    return &Pool{}, nil
}

// SSRRuntime extends JSRuntime with streaming RSC rendering.
// Implement core.SSRRuntime on top of your runtime type.
type SSRRuntime struct{ Runtime }

func (r *SSRRuntime) SetRequestContext(ctx map[string]any) error       { /* inject __REQUEST__ */ return nil }
func (r *SSRRuntime) CallRenderFunction(export, propsJSON string) (core.StreamHandle, error) { return nil, nil }
func (r *SSRRuntime) DrainStream(h core.StreamHandle, w core.StreamWriter) (bool, error)     { return false, nil }
```

## Step 2 — Register via init()

**`engine/<name>/register.go`**

```go
//go:build <name>

package myengine

import "github.com/polagonow/pola/core"

func init() {
    core.RegisterEngine(func() core.JSEngine { return NewEngine() })
}
```

## Step 3 — Register a polyfill fixture for tests

**`test/vm/<name>.go`**

```go
package vm

import (
    "testing"

    "github.com/polagonow/pola/engine/polyfill"
    "github.com/polagonow/pola/test/fixture"
    myengine "github.com/polagonow/pola/engine/<name>"
)

func init() {
    fixture.RegisterPolyfillVM("<name>", func(_ *testing.T) fixture.PolyfillFixture {
        rt := myengine.NewRuntime() // create a bare runtime instance
        return &myPolyfillFixture{rt: rt}
    })
}

type myPolyfillFixture struct{ rt *myengine.Runtime }

func (f *myPolyfillFixture) Enable() error {
    reg := polyfill.DefaultRegistry()
    for _, src := range reg.Get(
        polyfill.MicrotaskQueue,
        polyfill.TextEncoding,
        polyfill.MessageChannel,
        polyfill.ReadableStream,
        polyfill.AbortController,
        polyfill.WebpackRequire,
    ) {
        if err := f.rt.Eval(src.Source); err != nil {
            return err
        }
    }
    return nil
}

func (f *myPolyfillFixture) Eval(src string) error {
    _, err := f.rt.Eval(src)
    return err
}
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
go build -tags "<name> esbuild react nextjs" ./...
go test -tags "<name> esbuild react nextjs" -v ./engine/polyfill/...
go test -tags "<name> esbuild react nextjs" -v ./test/e2e/...
```

The e2e polyfill suite runs automatically for every registered VM fixture.

---
name: add-renderer
description: Add a new renderer to the GoJSX framework. Use when asked to add, integrate, or wire up a new UI renderer such as Vue, Svelte, Solid, or any alternative to React for server-side rendering.
---

A renderer takes the output of the VM (a streaming handle) and writes the HTTP
response. It implements `framework.Renderer`. Renderers also register a
`StreamProtocol` (which controls the `Content-Type` and how the stream is drained)
and optionally a `Discoverer`, `EntryGenerator`, `RouteBuilder`, and `HTMLShell`.

## Files to create

| File | Purpose |
|------|---------|
| `render/<name>/renderer.go` | `Renderer` implementation |
| `render/<name>/protocol.go` | `StreamProtocol` implementation |
| `render/<name>/register.go` | `init()` calling `framework.RegisterDefaults` |
| `render/<name>_render.go` | Build-tag import file (root `render/` package) |
| `test/combo/esbuild_<name>.go` | Test combo: esbuild bundler × new renderer |

---

## Step 1 — Implement `framework.Renderer`

**`render/<name>/renderer.go`** — full interfaces in `framework/interfaces.go`, types in `framework/contract/contract.go`

```go
package myrenderer

import (
    "context"

    "gojsx/framework"
    "gojsx/framework/contract"
)

// VMRenderer implements framework.Renderer.
type VMRenderer struct {
    pool         framework.VMPool
    protocol     framework.StreamProtocol
    globalBridge contract.BridgeConfig
}

// NewVMRenderer creates a Renderer backed by pool and protocol.
func NewVMRenderer(
    pool framework.VMPool,
    protocol framework.StreamProtocol,
    globalBridge contract.BridgeConfig,
) *VMRenderer {
    return &VMRenderer{pool: pool, protocol: protocol, globalBridge: globalBridge}
}

// Render implements framework.Renderer.
func (r *VMRenderer) Render(_ context.Context, w framework.StreamWriter, opts contract.RenderOpts) error {
    vm := r.pool.Acquire()
    defer r.pool.Release(vm)

    if err := vm.SetRequestContext(opts.RequestContext); err != nil {
        return err
    }

    bridge := r.globalBridge.Context
    if opts.Bridge != nil {
        bridge = opts.Bridge.Context
    }
    if err := vm.SetBridgeFunctions(bridge); err != nil {
        return err
    }

    handle, err := vm.CallRenderFunction(opts.ExportName, /* propsJSON */)
    if err != nil {
        return err
    }

    return r.protocol.Drain(vm, handle, w)
}
```

## Step 2 — Implement `framework.StreamProtocol`

**`render/<name>/protocol.go`**

```go
package myrenderer

import (
    "net/http"

    "gojsx/framework"
)

// Protocol implements framework.StreamProtocol.
type Protocol struct{}

func (p *Protocol) ContentType() string { return "text/x-myformat" }

func (p *Protocol) IsStreamingRequest(r *http.Request) bool {
    return r.Header.Get("Content-Type") == "text/x-myformat"
}

func (p *Protocol) Drain(vm framework.VM, handle framework.StreamHandle, w framework.StreamWriter) error {
    // pull chunks from the VM and write to w
    return nil
}
```

## Step 3 — Register global defaults

**`render/<name>/register.go`**

```go
package myrenderer

import (
    "gojsx/framework"
    "gojsx/framework/contract"
)

func init() {
    framework.RegisterDefaults(framework.Defaults{
        RendererFactory: func(pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig) framework.Renderer {
            return NewVMRenderer(pool, protocol, bridge)
        },
        StreamProtocol: func() framework.StreamProtocol { return &Protocol{} },
        // Register Discoverer, EntryGenerator, RouteBuilder, HTMLShell if needed.
    })
}
```

## Step 4 — Add the build-tag import file

**`render/<name>_render.go`** (in the root `render/` package):

```go
//go:build myrenderer

package render

import _ "gojsx/render/<name>"
```

## Step 5 — Add a test combo

**`test/combo/esbuild_<name>.go`**

```go
package combo

import (
    esbuild "gojsx/bundler/esbuild"
    "gojsx/framework"
    "gojsx/framework/contract"
    myrenderer "gojsx/render/<name>"
    "gojsx/test/fixture"
)

func init() { fixture.RegisterBundlerRenderer(&esbuildMyRendererCombo{}) }

type esbuildMyRendererCombo struct{}

func (c *esbuildMyRendererCombo) BundlerName() string  { return "esbuild" }
func (c *esbuildMyRendererCombo) RendererName() string { return "<name>" }

func (c *esbuildMyRendererCombo) NewBundler() framework.Bundler { return esbuild.NewBundler() }

func (c *esbuildMyRendererCombo) NewRendererFactory() func(framework.VMPool, framework.StreamProtocol, contract.BridgeConfig) framework.Renderer {
    return func(pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig) framework.Renderer {
        return myrenderer.NewVMRenderer(pool, protocol, bridge)
    }
}
```

No change to `test/e2e/ssr_rendering_test.go` needed.

## Key framework interfaces

All live in `framework/interfaces.go`:

- `framework.Renderer` — `Render(ctx, StreamWriter, RenderOpts) error`
- `framework.StreamProtocol` — `Drain`, `ContentType`, `IsStreamingRequest`
- `framework.StreamWriter` — `WriteRaw([]byte) (int, error)` + `Flush()`
- `framework.VMPool` — `Acquire() VM` + `Release(VM)`

## Verify

```
go build -tags "goja esbuild myrenderer" ./...
go test -v -timeout 120s ./test/e2e/... # "<vm>:<name>:esbuild" combos appear
```

Tests calling `fixture.ForEachReactApp` skip the new renderer (it's not "react").
Add renderer-specific suites using `fixture.ForEachApp` filtered by `f.Renderer() == "<name>"`.

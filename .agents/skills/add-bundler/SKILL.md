---
name: add-bundler
description: Add a new bundler to the GoJSX framework. Use when asked to add, integrate, or wire up a new JS bundler such as Vite, Rollup, Parcel, Webpack, or any other build tool.
---

A bundler takes source files from an app directory and produces a server-side JS
bundle (run inside the VM) plus client-side assets (served to the browser).
It implements `framework.Bundler`.

## Files to create

| File | Purpose |
|------|---------|
| `bundler/<name>/bundler.go` | `Bundler` implementation |
| `bundler/<name>/register.go` | `init()` that calls `framework.RegisterDefaults` |
| `bundler/<name>_bundler.go` | Build-tag import file (root `bundler/` package) |
| `test/combo/<name>_react.go` | Test combo: new bundler × React renderer |

---

## Step 1 — Implement `framework.Bundler`

**`bundler/<name>/bundler.go`** — full types in `framework/contract/contract.go`

```go
package mybundler

import "gojsx/framework/contract"

// Bundler implements framework.Bundler.
type Bundler struct{}

func NewBundler() *Bundler { return &Bundler{} }

func (b *Bundler) Bundle(input contract.BundleInput) (contract.BundleOutput, error) {
    // input fields:
    //   AppDir                 string   — root of the user's app
    //   OutDir                 string   — write client assets here
    //   AssetsURLPath          string   — URL prefix for client assets
    //   ClientEntry            string   — browser bootstrap entry point
    //   ClientComponents       []string — "use client" files to expose
    //   Dev                    bool     — enable source maps / watch mode
    //   ServerEntryContent     string   — pre-generated server entry JS
    //   ServerBundleConditions []string — package.json export conditions

    return contract.BundleOutput{
        ServerBundle:   []byte("/* compiled server JS */"),
        ClientManifest: map[string]contract.ClientComponent{},
    }, nil
}
```

## Step 2 — Register global defaults

**`bundler/<name>/register.go`**

```go
package mybundler

import "gojsx/framework"

func init() {
    framework.RegisterDefaults(framework.Defaults{
        Bundler: func() framework.Bundler { return &Bundler{} },
    })
}
```

## Step 3 — Add the build-tag import file

**`bundler/<name>_bundler.go`** (in the root `bundler/` package):

```go
//go:build mybundler && !embed

package bundler

import _ "gojsx/bundler/<name>"
```

The `!embed` constraint is intentional: when building with `embed`, assets are
pre-compiled into the binary and the bundler is not needed at runtime.

## Step 4 — Add a test combo

Create one file per bundler×renderer pair you want e2e-tested. Adding this file
is all that's needed — every registered VM automatically runs against it.

**`test/combo/<name>_react.go`**

```go
package combo

import (
    mybundler "gojsx/bundler/<name>"
    "gojsx/framework"
    "gojsx/framework/contract"
    react "gojsx/render/react"
    _ "gojsx/render/react/discovery/nextjs"
    _ "gojsx/render/react/shell"
    "gojsx/test/fixture"
)

func init() { fixture.RegisterBundlerRenderer(&mybundlerReactCombo{}) }

type mybundlerReactCombo struct{}

func (c *mybundlerReactCombo) BundlerName() string  { return "<name>" }
func (c *mybundlerReactCombo) RendererName() string { return "react" }

func (c *mybundlerReactCombo) NewBundler() framework.Bundler { return mybundler.NewBundler() }

func (c *mybundlerReactCombo) NewRendererFactory() func(framework.VMPool, framework.StreamProtocol, contract.BridgeConfig) framework.Renderer {
    return func(pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig) framework.Renderer {
        return react.NewVMRenderer(pool, protocol, bridge)
    }
}
```

No change to `test/e2e/ssr_rendering_test.go` — the file is in the `test/combo`
package already imported there.

## Verify

```
go build -tags "goja mybundler react" ./...
go test -v -timeout 120s ./test/e2e/... # "<vm>:react:<name>" combos appear
```

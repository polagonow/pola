---
name: add-vm
description: Add a new JavaScript engine (VM) to the GoJSX framework. Use when asked to add, integrate, or wire up a new JS runtime such as goja, v8go, sobek, quickjs, or any other engine.
---

A VM implementation wires a JavaScript runtime into the `framework.VM` interface
so the framework can execute React Server Components inside it.

## Files to create

| File | Purpose |
|------|---------|
| `vm/<name>/vm.go` | `VMFactory` + `VM` implementation |
| `vm/<name>/register.go` | `init()` that calls `framework.RegisterDefaults` |
| `vm/<name>/polyfill/polyfill.go` | VM-specific polyfill glue |
| `vm/<name>_vm.go` | Build-tag import file (root `vm/` package) |
| `test/vm/<name>.go` | `VMFixture` registration for the test suite |

---

## Step 1 — Implement `framework.VM` and `framework.VMFactory`

**`vm/<name>/vm.go`** — full interface signatures in `framework/interfaces.go`

```go
package myvm

import (
    "gojsx/framework"
    "gojsx/framework/contract"
)

// VMFactory implements framework.VMFactory.
type VMFactory struct{ /* compiled program */ }

func NewVMFactory(bundle []byte) (*VMFactory, error) {
    // compile/load the bundle into the runtime
    return &VMFactory{}, nil
}

func (f *VMFactory) New(bridge contract.BridgeConfig) (framework.VM, error) {
    return &VM{}, nil
}

// VM implements framework.VM.
type VM struct{ /* per-request state */ }

func (v *VM) SetRequestContext(ctx map[string]any) error                             { return nil }
func (v *VM) SetBridgeFunctions(funcs map[string]contract.GoFunc) error              { return nil }
func (v *VM) CallRenderFunction(export, propsJSON string) (framework.StreamHandle, error) { return nil, nil }
func (v *VM) ClearState() error                                                      { return nil }
```

Rules:
- `New` must call `mypolyfill.Enable(...)` **before** running the bundle.
- `SetBridgeFunctions` must wrap each Go func in a JS Promise (so `await` works).
- `ClearState` resets `__REQUEST__`, `__JSI__`, and any stream handles for pool reuse.

## Step 2 — Register global defaults

**`vm/<name>/register.go`**

```go
package myvm

import "gojsx/framework"

func init() {
    framework.RegisterDefaults(framework.Defaults{
        VMFactory: func(bundle []byte) (framework.VMFactory, error) {
            return NewVMFactory(bundle)
        },
    })
}
```

## Step 3 — Wire polyfills

**`vm/<name>/polyfill/polyfill.go`**

```go
package polyfill

import "gojsx/vm/polyfill"

func Enable(rt *MyRuntime) error {
    return polyfill.Load(&runner{rt: rt})
}

type runner struct{ rt *MyRuntime }

func (r *runner) RunScript(src, name string) error { return r.rt.Eval(src) }
```

Each VM needs polyfills for the Web APIs React Server Components rely on:
- `queueMicrotask` / `__drainMicrotasks__`
- `TextEncoder` / `TextDecoder`
- `MessageChannel`
- `ReadableStream`
- `AbortController`
- Webpack `require` shim

See `vm/goja/polyfill/` for the reference implementation.

## Step 4 — Add the build-tag import file

**`vm/<name>_vm.go`** (in the root `vm/` package):

```go
//go:build myvm

package vm

import _ "gojsx/vm/<name>"
```

## Step 5 — Register the VMFixture for tests

**`test/vm/<name>.go`**

```go
package vm

import (
    "testing"

    "gojsx/framework"
    "gojsx/test/fixture"
    myvm "gojsx/vm/<name>"
    mypolyfill "gojsx/vm/<name>/polyfill"
)

func init() { fixture.RegisterVM(&myVMFixture{}) }

type myVMFixture struct{}

func (f *myVMFixture) VMName() string { return "<name>" }
func (f *myVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
    return func(b []byte) (framework.VMFactory, error) { return myvm.NewVMFactory(b) }
}
func (f *myVMFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
    rt := /* create runtime instance */
    t.Cleanup(rt.Close)
    return &myPolyfillFixture{rt: rt}
}

type myPolyfillFixture struct{ rt *somelib.Runtime }

func (f *myPolyfillFixture) Enable() error         { return mypolyfill.Enable(f.rt) }
func (f *myPolyfillFixture) Eval(src string) error { _, err := f.rt.RunString(src); return err }
```

## Verify

```
go build -tags "myvm esbuild react" ./...
go test -v ./vm/polyfill/...            # polyfill tests under "<name>" sub-test
go test -v -timeout 120s ./test/e2e/... # e2e: "<name>:react:esbuild" combos appear
```

The e2e tests automatically run the new VM against every registered
bundler+renderer combo with no further changes needed.

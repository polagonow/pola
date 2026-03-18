---
name: add-polyfill
description: Add a new JavaScript Web API polyfill to the Pola engine layer. Use when asked to polyfill, implement, or add browser APIs (TextEncoder, MessageChannel, ReadableStream, queueMicrotask, AbortController, etc.) for server-side JS execution.
---

Polyfills live in `engine/polyfill/polyfill.go` as inline Go constants.
They are registered in `DefaultRegistry()` with a `core.PolyfillID` key.
Each engine declares which polyfills it needs via `RequiredPolyfills()` —
the pipeline injects only those at runtime startup.

Polyfill tests live in `engine/polyfill/` and run against every registered VM
via `fixture.ForEachVM`.

## Step 1 — Add the JS source and register the ID

**`engine/polyfill/polyfill.go`** — add a new constant + register it:

```go
// New well-known ID (add to the const block):
const MyPolyfill core.PolyfillID = "my-polyfill"

// JS source (add as a Go const — keep it ES5 for broadest engine compatibility):
const myPolyfillSrc = `(function () {
    if (typeof globalThis.myAPI !== 'undefined') { return; }
    // Globals added: myAPI
    // Depends on: (list earlier polyfills this one relies on)
    globalThis.myAPI = {
        doThing: function () { return "expected"; },
    };
})();`

// Register it in DefaultRegistry() (inside the existing function):
func DefaultRegistry() core.PolyfillRegistry {
    reg := NewRegistry()
    // ... existing registrations ...
    reg.Register(core.PolyfillSource{ID: MyPolyfill, Source: myPolyfillSrc})
    return reg
}
```

### Rules for the JS source

- Wrap in an IIFE — set only intentional globals on `globalThis`.
- Guard with `if (typeof globalThis.x !== 'undefined') { return; }` for idempotency.
- ES5 only — engines like qjs or v8go may not support modern syntax.
- Comment the globals this polyfill adds and any polyfills it depends on.

### Current polyfills and load order

| ID constant | Installs | Must come after |
|-------------|----------|-----------------|
| `MicrotaskQueue` | `queueMicrotask`, `__drainMicrotasks__` | — |
| `TextEncoding` | `TextEncoder`, `TextDecoder` | — |
| `MessageChannel` | `MessageChannel`, `MessagePort` | `MicrotaskQueue` |
| `ReadableStream` | `ReadableStream` + controller | `MicrotaskQueue`, `TextEncoding` |
| `AbortController` | `AbortController`, `AbortSignal` | — |
| `WebpackRequire` | `__webpack_require__` shim | — |
| `Promise` | `Promise` | `MicrotaskQueue` |

## Step 2 — Declare it in the engines that need it

In each `engine/<name>/<name>.go`, add the new ID to `RequiredPolyfills()`:

```go
func (e *Engine) RequiredPolyfills() []core.PolyfillID {
    return []core.PolyfillID{
        polyfill.MicrotaskQueue,
        // ... existing ...
        polyfill.MyPolyfill,   // ← add here
    }
}
```

Engines with native support for an API should NOT include the polyfill.

## Step 3 — Write the test

**`engine/polyfill/<name>_test.go`**

```go
package polyfill_test

import (
    "testing"

    "github.com/polagonow/pola/test/fixture"
    _ "github.com/polagonow/pola/test/vm" // registers all VM fixtures
)

func TestMyPolyfillBasic(t *testing.T) {
    fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
        // Enable() already installed all polyfills declared in the VM's RequiredPolyfills.
        if err := f.Eval(`
            var result = myAPI.doThing();
            if (result !== "expected") throw new Error("got: " + result);
        `); err != nil {
            t.Fatal(err)
        }
    })
}

func TestMyPolyfillEdgeCase(t *testing.T) {
    fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
        // Each ForEachVM call gets a fresh context — no state from previous tests.
        if err := f.Eval(`
            if (typeof myAPI === 'undefined') throw new Error("myAPI not installed");
        `); err != nil {
            t.Fatal(err)
        }
    })
}
```

### Skipping specific VMs

```go
fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
    // The VM name is accessible via the fixture name used in RegisterPolyfillVM.
    // To skip v8go (which has this API natively):
    // Check test name: t.Name() contains the vm name as a sub-test segment.
    if err := f.Eval(`...`); err != nil {
        t.Skip("engine does not support this API natively")
    }
})
```

## Verify

```
go test -tags "goja esbuild react nextjs" -v ./engine/polyfill/...
```

All VM fixtures registered in `test/vm/` run automatically.

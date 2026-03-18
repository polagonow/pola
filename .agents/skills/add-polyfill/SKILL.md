---
name: add-polyfill
description: Add a new JavaScript Web API polyfill to the GoJSX VM layer. Use when asked to polyfill, implement, or add browser APIs (TextEncoder, MessageChannel, ReadableStream, queueMicrotask, AbortController, etc.) for server-side JS execution.
---

Polyfill tests live in `vm/polyfill/` and verify that a JS polyfill behaves correctly
across **every registered VM engine**. Every polyfill is two files: a `.js`
implementation loaded into every VM at startup, and a Go test file.

`polyfill.Load` (`vm/polyfill/impls.go`) uses `//go:embed *.js` and walks all `.js`
files in alphabetical order, so the `NN_` prefix controls load order. Polyfills
that depend on earlier ones (e.g. `messagechannel` needs `queueMicrotask`) must
have a higher number.

## Step 1 — Write the JS polyfill

**`vm/polyfill/08_mypolyfill.js`**

```js
// Globals added: myPolyfill
// Depends on: (list any earlier polyfill names here)
(function () {
    globalThis.myPolyfill = {
        doThing: function () { return "expected"; },
        edgeCase: function (v) { return v == null ? 0 : v; },
    };
})();
```

Rules:
- Wrap everything in an IIFE — no globals except the ones you intentionally set on `globalThis`.
- Comment the globals this file adds and any polyfills it depends on (load-order guard).
- ES5 only — engines like qjs or v8go may not support modern syntax.

Current load order: `01_microtask` → `02_textencoding` → `03_messagechannel` →
`04_readablestream` → `05_webpackrequire` → `06_abortcontroller` → `07_promise`

## Step 2 — Write the test file

**`vm/polyfill/08_mypolyfill_test.go`**

```go
package polyfill_test

import (
    "testing"

    fixture "gojsx/test/fixture"
    _ "gojsx/test/vm"
)

func TestMyPolyfillBasic(t *testing.T) {
    fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
        if err := f.Eval(`
            // polyfills are already installed by Enable() via the import above.
            var result = myPolyfill.doThing();
            if (result !== "expected") throw new Error("got: " + result);
        `); err != nil {
            t.Fatal(err)
        }
    })
}

func TestMyPolyfillEdgeCase(t *testing.T) {
    fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
        if err := f.Eval(`
            // Each ForEachVM call gets a fresh context — no state leaks.
            var x = myPolyfill.edgeCase(null);
            if (x !== 0) throw new Error("expected 0, got " + x);
        `); err != nil {
            t.Fatal(err)
        }
    })
}
```

## How `ForEachVM` works

`fixture.ForEachVM` calls `f.Enable()` before passing the fixture to your function.
`Enable()` installs all polyfills for that VM via the VM's own polyfill registry.
Each sub-test gets a **fresh** JS context (no state from previous tests).

## Numbering convention

Files are prefixed `NN_` to control test discovery order when running with `-v`.
Current files: `01_microtask`, `02_textencoding`, `03_messagechannel`,
`04_readablestream`, `05_webpackrequire`, `06_abortcontroller`, `07_promise`.
Pick the next available prefix or omit it for standalone tests.

## If your polyfill only works on some VMs

```go
fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
    if f.(interface{ VMName() string }).VMName() == "v8go" {
        t.Skip("v8go has native support, polyfill not installed")
    }
    // ...
})
```

Or gate at the `PolyfillFixture` level by returning an error from `Enable`.

## Verify

```
go test -v ./vm/polyfill/...
```

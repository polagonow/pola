---
name: add-e2e-test
description: Add a new E2E test suite or test case to the Pola framework. Use when asked to write, add, or implement end-to-end tests, integration tests, or HTTP-level tests against the rendered app.
---

E2E tests live in `test/e2e/suite/` (one file per feature area) and run against
every registered VM × bundler+renderer combo automatically via `fixture.ForEachApp`.
Subsystem-specific suites live next to it: `test/e2e/renderer/react/` (React-only,
Flight/error-boundary/layout), `test/e2e/bundler/` (client bundle), and
`test/e2e/engine/` (polyfills, via `fixture.ForEachVM`).

## Pattern

Each suite file exports a single `RunXxxTests(t *testing.T)` function that groups
related sub-tests. Tests use helper functions from `github.com/polagonow/pola/test/fixture` to make HTTP
requests against the fully-built app.

## Step 1 — Create the suite file

**`test/e2e/suite/my_feature_suite.go`**

```go
package suite

import (
    "strings"
    "testing"

    "github.com/polagonow/pola/test/fixture"
)

// RunMyFeatureTests verifies [describe what this suite tests].
func RunMyFeatureTests(t *testing.T) {
    t.Helper()

    t.Run("SomeAssertion", func(t *testing.T) {
        // ForEachReactApp: runs for every VM, but only React renderer combos.
        // Use ForEachApp instead if the test is renderer-agnostic.
        fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
            body := fixture.RSC(t, f, "/some/path")
            if !strings.Contains(body, "expected content") {
                t.Errorf("expected content not found in: %s", body)
            }
        })
    })

    t.Run("AnotherAssertion", func(t *testing.T) {
        fixture.ForEachReactApp(t, func(t *testing.T, f fixture.AppFixture) {
            body := fixture.Page(t, f, "/")
            if !strings.Contains(body, `id="root"`) {
                t.Error("missing mount point")
            }
        })
    })
}
```

## Step 2 — Register in the test entry point

**`test/e2e/rendering_test.go`** — add one line:

```go
func TestMyFeature(t *testing.T) { suite.RunMyFeatureTests(t) }
```

## HTTP helpers (from `github.com/polagonow/pola/test/fixture`)

| Helper | What it does |
|--------|-------------|
| `fixture.RSC(t, f, path)` | GET with `Content-Type: text/x-component`; fails on non-200 |
| `fixture.RSCAny(t, f, path)` | RSC request, returns `(status, body)`, never fails |
| `fixture.Page(t, f, path)` | Normal HTML GET; fails on non-200 |
| `fixture.PageAny(t, f, path)` | HTML GET, returns `(status, body)`, never fails |
| `fixture.FlightTree(t, body)` | Parses root `0:` Flight row as JSON |
| `fixture.FlightContains(body, s)` | `strings.Contains` over the flight body |
| `f.GetApp(t)` | Returns the built `*core.App` for direct `ServeHTTP` calls |
| `fixture.SharedInjector()` | The runtime injector carrying the blog test data (used when wiring custom combos) |
| `fixture.AppDir` | Path to the shared fixture app (`examples/blog-e2e-react/web`) |

## Fixture iteration

| Function | When to use |
|----------|-------------|
| `fixture.ForEachApp` | Test applies to all bundler+renderer combinations |
| `fixture.ForEachReactApp` | Test is React-specific (uses RSC / Flight / React HTML output) |

## Filtering on bundler/renderer inside a test

```go
fixture.ForEachApp(t, func(t *testing.T, f fixture.AppFixture) {
    if f.BundlerName() != "esbuild" {
        t.Skipf("skipping: test requires esbuild, got %s", f.BundlerName())
    }
    // ...
})
```

## Verify

```
go test -v -run TestMyFeature ./test/e2e/...
```

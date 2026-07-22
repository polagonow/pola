---
name: add-web-framework
description: Add a new HTTP web framework adapter to the Pola framework behind core.WebFramework. Use when asked to add, integrate, or wire an HTTP router/framework such as Fiber, Iris, httprouter, or fasthttp alongside the existing std, gin, echo, and chi adapters.
---

Pola serves everything through a pluggable HTTP web framework. An adapter implements
`core.WebFramework` (`core/web.go`) and is provided to DI as a **factory**
(`core.WebFrameworkFactory = func() core.WebFramework`). Apps select it with the
Polafile `framework` attribute, the `--framework` CLI flag, or `POLA_WEB_FRAMEWORK`
(values today: `std` (default), `gin`, `echo`, `chi`).

## Files to create

| File | Purpose |
|------|---------|
| `webframework/<name>/<name>.go` | `core.WebFramework` implementation |
| `webframework/<name>/plugin.go` | `Plugin() core.Plugin` providing the factory |

---

## Step 1 — Implement `core.WebFramework`

**`webframework/<name>/<name>.go`** — reference: `webframework/chi/chi.go` (or
`webframework/std/std.go` for the zero-dependency baseline).

```go
package myfw

import (
    "net/http"

    "github.com/polagonow/pola/core"
)

type Framework struct{ /* underlying router */ }

func New() *Framework { return &Framework{} }

func (f *Framework) Name() string { return "<name>" }

// Handle registers a handler for method+pattern. Pola patterns use
// ":param" for a single segment and ":...rest" for a trailing catch-all —
// translate them to your router's syntax.
func (f *Framework) Handle(method, pattern string, h core.HandlerFunc) { /* ... */ }

// Use appends a global middleware in registration order.
func (f *Framework) Use(mw func(next http.Handler) http.Handler) { /* ... */ }

// Mount attaches a sub-handler under a path prefix (used for reserved
// endpoints like /_pola/assets, /_pola/image, /mcp).
func (f *Framework) Mount(prefix string, h http.Handler) { /* ... */ }

// Fallback sets the catch-all handler for unmatched routes (page rendering).
func (f *Framework) Fallback(h http.Handler) { /* ... */ }

// Handler returns the root http.Handler the server will serve.
func (f *Framework) Handler() http.Handler { /* ... */ return nil }
```

Semantics to preserve (see how `webframework/std` does it):

- Middleware added via `Use` must wrap **everything**, including `Mount`ed
  handlers and the `Fallback`.
- `Fallback` must only fire when no `Handle`/`Mount` pattern matched.
- Path params extracted from `:param` segments are delivered through
  `core.HandlerFunc`'s context — follow the existing adapters' param-passing
  convention exactly.

## Step 2 — Register via `Plugin()`

**`webframework/<name>/plugin.go`** — reference: `webframework/gin/plugin.go`

```go
package myfw

import "github.com/polagonow/pola/core"

// Plugin registers the <name> web framework as a factory in the DI container.
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "webframework:<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.WebFrameworkFactory](r, func() core.WebFramework { return New() })
        },
    }
}
```

Note it provides a **factory**, not an instance — the pipeline constructs the
framework when it wires the app.

## Step 3 — Expose it to the CLI

The CLI maps the Polafile `framework` value to a plugin import in the generated
wiring (`pola_plugins.go`) via `internal/autoload`. Add `<name>` wherever the
existing four are enumerated (search `internal/` and `cmd/` for `"chi"` to find
the switch/template) and to the `--framework` flag help text in
`internal/cli/{new,serve,build}.go` (`"HTTP web framework (std, gin, echo, chi)"`).

## Verify

```
go build ./...
go test ./webframework/...
```

Then scaffold an example app and run it on the new adapter:

```
pola new fw-test --api-only -y
cd fw-test && pola dev --framework <name>
```

Routes in `routes/`, reserved `/_pola/*` endpoints, and middleware (CSRF,
security headers) must all behave identically to `--framework std`.

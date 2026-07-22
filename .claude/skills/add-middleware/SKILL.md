---
name: add-middleware
description: Add a new HTTP middleware to the Pola framework. Use when asked to add, implement, or wire request middleware (auth guards, rate limiting, headers, logging, compression, tenant resolution, etc.) alongside the existing middleware packages (cors, csrf, health, ratelimit, securityheaders, logging, recovery, session, flash, jwt, requireauth, locale, compression).
---

A middleware implements `core.Middleware` — `Name() string` and
`Wrap(next http.Handler) http.Handler` — and lives in its own package under
`middleware/<name>/` with a `Plugin(opts ...Option) core.Plugin` that registers it
via `r.AddMiddleware(...)`. Global middleware wraps every request (pages, routes,
and `/_pola/*` endpoints); per-route middleware uses the separate
`core.RouteMiddleware` type attached from a route struct's `Middleware()` method.

## Files to create

| File | Purpose |
|------|---------|
| `middleware/<name>/<name>.go` | The middleware type + functional options |
| `middleware/<name>/plugin.go` | `Plugin(opts ...Option) core.Plugin` calling `r.AddMiddleware` |
| `middleware/<name>/<name>_test.go` | httptest-based unit test |

---

## Step 1 — Implement `core.Middleware`

**`middleware/<name>/<name>.go`** — reference: `middleware/cors/cors.go`

```go
package myname

import (
    "net/http"

    "github.com/polagonow/pola/core"
)

type middleware struct {
    cfg config
}

type config struct {
    // knobs with sensible dev-friendly defaults
}

// Option customizes the middleware.
type Option func(*config)

func WithSomething(v string) Option {
    return func(c *config) { /* ... */ }
}

// New returns the middleware with the given options applied.
func New(opts ...Option) core.Middleware {
    cfg := config{ /* defaults */ }
    for _, o := range opts {
        o(&cfg)
    }
    return &middleware{cfg: cfg}
}

func (m *middleware) Name() string { return "<name>" }

func (m *middleware) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // before …
        next.ServeHTTP(w, r)
        // after …
    })
}
```

Conventions (all existing middleware follow them):

- Functional options with safe defaults; no config structs in the public API.
- Short-circuit by writing the response and *not* calling `next`.
- Pass request-scoped values down via `context` (see `middleware/session`,
  `auth.WithUser` for the pattern); read them in handlers with a typed getter.

## Step 2 — Register via `Plugin()`

**`middleware/<name>/plugin.go`** — reference: `middleware/cors/plugin.go`

```go
package myname

import "github.com/polagonow/pola/core"

// Plugin returns the <name> middleware plugin. Register it directly:
//
//	pola.Use(myname.Plugin(myname.WithSomething("value")))
func Plugin(opts ...Option) core.Plugin {
    return core.PluginFunc{
        PluginName: "<name>",
        Fn: func(r *core.Registry) {
            r.AddMiddleware(New(opts...))
        },
    }
}
```

Middleware runs in registration order. If the middleware should be configurable
from `Polafile.hcl` (like `csrf`, `rate_limit`, `session`), add a config block in
`polafile/polafile.go` and teach `internal/autoload` to emit the plugin into the
generated `pola_plugins.go` when the block is enabled — mirror how the
`rate_limit` block maps to `middleware/ratelimit`.

## Per-route middleware

For guards that apply to specific HTTP routes rather than globally, provide a
`core.RouteMiddleware` constructor instead (reference: `auth.RouteMiddleware` in
`auth/route.go`). Apps attach it by implementing `Middleware() []core.RouteMiddleware`
on a route struct in `routes/`.

## Step 3 — Test

```go
func TestMyName(t *testing.T) {
    h := New().Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
    // assert on rec
}
```

## Verify

```
go build ./...
go test ./middleware/<name>/...
mage check
```

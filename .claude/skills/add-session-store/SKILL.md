---
name: add-session-store
description: Add a new server-side session store to the Pola framework backed by gorilla/sessions. Use when asked to add, implement, or wire a session store/backend (Postgres, MySQL, DynamoDB, Memcached, Mongo, etc.) alongside the existing session/redis and session/xorm stores.
---

Session stores are plain `gorilla/sessions.Store` implementations — Pola defines
no interface of its own. A store package lives under `session/<name>/` and
exposes a functional-options constructor returning `sessions.Store`, plus a
`MustNew` panic wrapper used by generated wiring. The store is consumed by the
session middleware (`middleware/session/`), which is the actual `core.Plugin`:
`session.Plugin(session.WithStore(store))` registers it via `r.AddMiddleware`.
Prefer wrapping an existing gorilla store library (as redis wraps
`rbcervilla/redisstore/v9` and xorm wraps `lafriks/xormstore`) over writing
Get/New/Save from scratch.

## Files to create / edit

| File | Purpose |
|------|---------|
| `session/<name>/<name>.go` | Options + `NewStore` + `MustNew` (new) |
| `session/<name>/<name>_test.go` | Unit tests (new — see Step 4) |
| `internal/autoload/pluginimports/_templates/plugins_go.tmpl` | Template branch so Polafile `store = "<name>"` works (edit) |

## Step 1 — Implement the store package

The interface to satisfy (github.com/gorilla/sessions v1.4.0):

```go
type Store interface {
	Get(r *http.Request, name string) (*sessions.Session, error)
	New(r *http.Request, name string) (*sessions.Session, error)
	Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error
}
```

**`session/<name>/<name>.go`** — follow `session/redis/redis.go` exactly
(options struct is private; every knob is a `With*` option; constructor fails
fast on connectivity):

```go
package myname

import (
	"fmt"

	"github.com/gorilla/sessions"
	// + the third-party gorilla store lib you are wrapping
)

type Option func(*config)

type config struct {
	dsn string // whatever the backend needs
}

func WithDSN(dsn string) Option {
	return func(c *config) { c.dsn = dsn }
}

func NewStore(opts ...Option) (sessions.Store, error) {
	cfg := &config{ /* defaults */ }
	for _, o := range opts {
		o(cfg)
	}
	store, err := /* wrap backend lib */
	if err != nil {
		return nil, fmt.Errorf("session/myname: create store: %w", err)
	}
	return store, nil
}

func MustNew(opts ...Option) sessions.Store {
	s, err := NewStore(opts...)
	if err != nil {
		panic(err)
	}
	return s
}
```

Signature notes from the real stores: `session/redis` is
`NewStore(opts ...Option) (sessions.Store, error)` with `WithAddr`,
`WithPassword`, `WithDB`, `WithKeyPrefix` (default addr `localhost:6379`, prefix
`session:`); `session/xorm` takes the cookie-signing keys first —
`NewStore(keyPairs []byte, opts ...Option)` — and starts a
`store.PeriodicCleanup` goroutine. Match whichever shape your backend needs.

Values are gob-encoded by gorilla stores. `middleware/session/session.go`
already registers `uint`, `uint64`, `map[string]string`, `map[string]any`; any
other value type an app stores needs its own `gob.Register`.

## Step 2 — Registration (the middleware is the plugin)

There is no per-store plugin. `middleware/session/plugin.go` is the single
`core.Plugin` (shape: `core.PluginFunc{PluginName: "session", Fn: func(r
*core.Registry) { r.AddMiddleware(New(opts...)) }}`), and the store is injected
with `session.WithStore(store sessions.Store)`. If no store is given, the
middleware falls back to a `sessions.NewCookieStore` with a random per-process
key. The middleware handles cookie options, save-on-first-write, flash messages,
and `session.Regenerate` (which calls `store.New` to rotate the session ID) —
your store must support `New` returning a fresh session for that to work.

## Step 3 — How an app enables it

Polafile block (`polafile/polafile.go`, `Session` struct — attributes `enabled`,
`store`, `max_age`, `host`, `port`, `password`, `db`, `dsn`, plus per-env
blocks; env overrides `POLA_SESSION_STORE`, `POLA_SESSION_HOST`, etc.
`SessionStore()` defaults to `"cookie"`):

```hcl
session {
  store = "myname"
  dsn   = "..."
}
```

Store selection is hard-coded in
`internal/autoload/pluginimports/_templates/plugins_go.tmpl`: the import block
has `{{- if eq .SessionStore "redis"}}` / `"xorm"` branches (aliases
`sessionredis` / `sessionxorm`), and the `PolaPlugins` list emits e.g.

```go
session.Plugin(session.WithStore(sessionredis.MustNew(sessionredis.WithAddr(...)))),
```

**Any unrecognized store name falls through to the plain cookie store** — add an
`else if eq .SessionStore "myname"` branch in both places (imports ~line 38,
plugin call ~line 175), mapping the Polafile `host`/`port`/`password`/`db`/`dsn`
fields to your `With*` options via `polaenv.String("POLA_SESSION_...", ...)`.

Manual (non-autoload) apps:

```go
pola.Use(session.Plugin(
	session.WithStore(sessionmyname.MustNew(sessionmyname.WithDSN("..."))),
))
```

## Step 4 — Tests

There is no standard test pattern for session stores — neither `session/redis/`
nor `session/xorm/` ships tests. Add a unit test alongside
(`session/<name>/<name>_test.go`): at minimum cover option application and
constructor error paths; a Get/New/Save round-trip against a real backend should
`t.Skip` unless an env var (e.g. `MYNAME_DSN`) is set, since `NewStore` pings
the backend and would fail in CI.

## Verify

```
go build ./...
go vet ./session/...
go test ./session/... ./middleware/session/...
```

If you touched the autoload template, run `go test ./internal/autoload/...` and
scaffold an example app with `session { store = "myname" }` to confirm the
generated `pola_plugins.go` compiles.

---
name: add-cache
description: Add a new cache backend to the Pola framework behind the core.Cache interface. Use when asked to add, implement, or wire a cache adapter/backend (Redis, Memcached, BoltDB, groupcache, etc.), or to finish the cache/redis stub.
---

Cache backends implement `core.Cache` (`core/interfaces.go`) and live in their own
package under `cache/<adapter>/`. Registration is a plain `Plugin() core.Plugin`
function that calls `core.ProvideValue[core.Cache]` — no `init()` registries, no
build tags. The package name doubles as the Polafile `adapter` value: generated
`pola_plugins.go` imports `cache/<adapter>` and calls `<adapter>.Plugin()`.

Reference implementation: `cache/memory/` (complete, LRU via hashicorp/golang-lru).
Note: `cache/redis/` is a stub — every method returns `ErrNotImplemented` and it
has **no plugin.go**, so it cannot currently be selected by autoload.

## Files to create

| File | Purpose |
|------|---------|
| `cache/<adapter>/<adapter>.go` | The `core.Cache` implementation |
| `cache/<adapter>/plugin.go` | `Plugin() core.Plugin` DI registration |
| `cache/<adapter>/<adapter>_test.go` | Unit tests (mirror `cache/memory/memory_test.go`) |

## Step 1 — Implement core.Cache

Exact interface (`core/interfaces.go`):

```go
type Cache interface {
	Name() string
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, opts CacheOptions) error
	Delete(ctx context.Context, key string) error
	Invalidate(ctx context.Context, prefix string) error
	Clear(ctx context.Context) error
}
```

`core.CacheOptions` (`core/types.go`) has one field: `TTL time.Duration` — zero
means no expiry. Semantics to preserve (see `cache/memory/memory.go`):

- `Get` returns `(nil, false, nil)` on miss or expired entry — a miss is not an error.
- `Invalidate(ctx, prefix)` deletes every key with that string prefix.
- `Name()` returns the adapter name (`"memory"`, `"redis"`, ...).
- Must be safe for concurrent use.

**`cache/<adapter>/<adapter>.go`** skeleton (adapted from memory):

```go
// Package myadapter provides a my-backend cache for Pola.
package myadapter

import (
	"context"
	"time"

	"github.com/polagonow/pola/core"
)

type Cache struct{ /* client/handle */ }

func New( /* config */ ) (*Cache, error) { /* connect */ return &Cache{}, nil }

// MustNew panics on error — used by generated plugin wiring.
func MustNew() *Cache { c, err := New(); if err != nil { panic(err) }; return c }

func (c *Cache) Name() string { return "myadapter" }
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error)          { ... }
func (c *Cache) Set(ctx context.Context, key string, val []byte, opts core.CacheOptions) error { ... }
func (c *Cache) Delete(ctx context.Context, key string) error                        { ... }
func (c *Cache) Invalidate(ctx context.Context, prefix string) error                 { ... }
func (c *Cache) Clear(ctx context.Context) error                                     { ... }
```

## Step 2 — Add plugin.go

Copy the exact shape of `cache/memory/plugin.go`:

```go
package myadapter

import "github.com/polagonow/pola/core"

// Plugin returns the myadapter cache plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "myadapter-cache",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Cache](r, MustNew())
		},
	}
}
```

`Plugin()` must take **no arguments** — autoload calls it niladically (see below).
If the backend needs connection config, read env vars inside `New`/`Plugin` via
`github.com/polagonow/pola/env` (e.g. `env.String("POLA_CACHE_HOST", "localhost")`).

## Step 3 — How an app enables it

Polafile block (`polafile/polafile.go`, `Cache` struct — attributes `enabled`,
`adapter`, `host`, `port`, `password`, `db`, plus per-env `env "prod" { ... }` blocks):

```hcl
cache {
  adapter = "myadapter"   # default is "memory"; "none" disables the cache plugin
}
```

`internal/autoload/pluginimports/_templates/plugins_go.tmpl` then emits:

```go
import "github.com/polagonow/pola/cache/myadapter"
// ...
myadapter.Plugin(),
```

No template edit is needed as long as the package directory matches the adapter
name. **Caveat:** the template calls `{{.Cache}}.Plugin()` with no args, so the
Polafile `host`/`port`/`password`/`db` attributes are *not* forwarded to cache
plugins today — read `POLA_CACHE_*` env vars yourself, or extend the template.

Manual (non-autoload) apps register it directly:

```go
pola.Use(myadapter.Plugin())
```

## Step 4 — Tests

Follow `cache/memory/memory_test.go`: external test package, plain stdlib
`testing`, exercising miss, set/get, delete, prefix invalidation, clear, and
`Name()`. Example assertion style:

```go
if err := c.Set(ctx, "k", []byte("v"), core.CacheOptions{}); err != nil { ... }
val, ok, err := c.Get(ctx, "k")
```

If the backend needs a live server, guard with an env var skip
(`t.Skip` unless e.g. `MYADAPTER_ADDR` is set) so `go test ./...` stays green.

## Verify

```
go build ./...
go vet ./cache/...
go test ./cache/...
```

Then in an example app: set `cache { adapter = "myadapter" }` in the Polafile and
run `pola dev` — the generated `pola_plugins.go` should import and register it.

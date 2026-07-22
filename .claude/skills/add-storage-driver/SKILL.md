---
name: add-storage-driver
description: Add a new file storage driver to the Pola framework behind the storage.Storage interface. Use when asked to add, implement, or wire a storage driver/backend (S3-native, GCS, Azure, MinIO, in-memory, SFTP, etc.) alongside the existing fs and rclone drivers.
---

Storage drivers implement `storage.Storage` (`storage/storage.go`) and live in
their own package under `storage/<driver>/`. Unlike caches, drivers have **no
per-driver plugin.go**: the generic `storage.Plugin(s Storage)` in
`storage/plugin.go` wraps any pre-constructed driver and registers it via
`core.ProvideValue[Storage]`. Existing drivers: `storage/fs/` (local disk) and
`storage/rclone/` (70+ cloud backends via the rclone library — check whether a
new backend is already covered by rclone before writing a native driver).

## Files to create / edit

| File | Purpose |
|------|---------|
| `storage/<driver>/<driver>.go` | The `storage.Storage` implementation (new) |
| `storage/<driver>/<driver>_test.go` | Unit tests (new — see Step 4) |
| `storage/storage.go` | Add a `StorageDriver` constant (edit) |
| `internal/autoload/pluginimports/_templates/plugins_go.tmpl` | Template branch so the Polafile can select it (edit) |

## Step 1 — Implement storage.Storage

Exact interface (`storage/storage.go`) — implementations must be safe for
concurrent use:

```go
type Storage interface {
	// Save writes content to the given path. Overwrites if it exists.
	Save(ctx context.Context, content io.Reader, path string) error
	// Stat returns metadata. Returns ErrNotExist if the file does not exist.
	Stat(ctx context.Context, path string) (*Stat, error)
	// Open returns a reader (caller closes). Returns ErrNotExist if missing.
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	// Delete removes the file at path.
	Delete(ctx context.Context, path string) error
	// List returns metadata for all files under path.
	List(ctx context.Context, path string) ([]*Stat, error)
}
```

Supporting types in the same file: sentinel `var ErrNotExist = errors.New("file
does not exist")` (must be returned by `Stat`/`Open`, and `List` in the fs
driver, for missing paths) and

```go
type Stat struct {
	ModifiedTime time.Time `json:"modified_time"`
	Size         int64     `json:"size"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	ContentType  string    `json:"content_type"`
}
```

**`storage/<driver>/<driver>.go`** skeleton, following `storage/fs/fs.go`
(config struct + `NewStorage` constructor; fs uses
`mime.TypeByExtension(filepath.Ext(name))` for `ContentType` and rejects paths
that escape the root):

```go
// Package mydriver implements a my-backend storage driver.
package mydriver

import (
	"context"
	"io"

	"github.com/polagonow/pola/storage"
)

type Storage struct{ /* client, bucket, ... */ }

type Config struct{ /* Root, Bucket, ... */ }

// NewStorage returns a new mydriver storage.
func NewStorage(cfg Config) *Storage { return &Storage{ /* ... */ } }

func (s *Storage) Save(ctx context.Context, content io.Reader, path string) error { ... }
func (s *Storage) Stat(ctx context.Context, path string) (*storage.Stat, error)   { ... } // storage.ErrNotExist on miss
func (s *Storage) Open(ctx context.Context, path string) (io.ReadCloser, error)   { ... } // storage.ErrNotExist on miss
func (s *Storage) Delete(ctx context.Context, path string) error                  { ... }
func (s *Storage) List(ctx context.Context, path string) ([]*storage.Stat, error) { ... }
```

(For comparison, rclone's constructor is `NewStorage(driver, location string) *Storage`.)

## Step 2 — Registration (no new plugin needed)

`storage/plugin.go` already provides the plugin — verify your driver satisfies
the interface and just pass an instance in:

```go
func Plugin(s Storage) core.Plugin {
	return core.PluginFunc{
		PluginName: "storage",
		Fn: func(r *core.Registry) {
			core.ProvideValue[Storage](r, s)
		},
	}
}
```

Also add a driver name constant to the `const` block in `storage/storage.go`
next to `Filesystem StorageDriver = "fs"` and `Rclone StorageDriver = "rclone"`.

## Step 3 — How an app enables it

Polafile block (`polafile/polafile.go`, `StorageConfig` — attributes `driver`,
`root`, `config_path`, plus per-env `env "prod" { ... }` blocks; env overrides
`POLA_STORAGE_DRIVER` / `POLA_STORAGE_ROOT` / `POLA_STORAGE_CONFIG_PATH`):

```hcl
storage {
  driver = "mydriver"
  root   = "uploads"
}
```

Autoload wiring is a hard-coded branch in
`internal/autoload/pluginimports/_templates/plugins_go.tmpl` (search
`StorageDriver`): `{{- if eq .StorageDriver "rclone"}}` imports
`storage/rclone`, **anything else falls through to `storage/fs`** — so a new
driver name silently becomes fs until you add an `else if` branch in both the
import block (~line 92) and the `storage.Plugin(...)` call site (~line 239).

Manual (non-autoload) apps skip the template entirely:

```go
import (
	"github.com/polagonow/pola/storage"
	"github.com/polagonow/pola/storage/mydriver"
)

pola.Use(storage.Plugin(mydriver.NewStorage(mydriver.Config{ /* ... */ })))
```

## Step 4 — Tests

There is no standard test pattern for storage drivers — neither `storage/fs/`
nor `storage/rclone/` ships tests. Add a unit test alongside
(`storage/<driver>/<driver>_test.go`) covering: Save then Stat/Open round-trip,
`storage.ErrNotExist` from `Stat`/`Open` on a missing path, Delete, and List.
Drivers needing external services should `t.Skip` unless a config env var is set.

## Verify

```
go build ./...
go vet ./storage/...
go test ./storage/...
```

If you touched the autoload template, also run `go test ./internal/autoload/...`
and scaffold/run an example app with `storage { driver = "mydriver" }` to check
the generated `pola_plugins.go`.

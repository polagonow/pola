# core

Core types, interfaces, and registry for the Pola framework.

This package defines the contracts that all plugins must satisfy. It has **zero external dependencies** — only the Go standard library.

## Contents

| File | Purpose |
|------|---------|
| `interfaces.go` | All plugin interfaces (`JSEngine`, `Renderer`, `Bundler`, `Router`, `FS`, `Cache`, …) |
| `types.go` | Shared value types (`RenderRequest`, `BundleInput`, `Route`, `CacheOptions`, …) |
| `registry.go` | `Registry`, `Config`, `App` — the top-level wiring types |
| `errors.go` | Sentinel errors |
| `globals/globals.go` | Well-known JS global names (`__REQUEST__`, `__DEPENDENCY_INJECTION__`, …) |
| `env/env.go` | `POLA_*` environment variable struct (parsed via `caarlos0/env`) |

## Key interfaces

```go
type JSEngine   interface { Name() string; NewRuntime(ctx) (JSRuntime, error); RequiredPolyfills() []PolyfillID }
type Renderer   interface { Name() string; FileExtensions() []string; Render(ctx, RenderRequest) (RenderResult, error); Capabilities() []Capability }
type Router     interface { Name() string; ScanRoutes(ctx, FS, appDir, exts) ([]Route, error); Resolve(ctx, path) (*Route, map[string]any) }
type Bundler    interface { Name() string; Build(ctx, BundleInput) (*BundleOutput, error); Watch(ctx, BundleInput, onChange) error }
type FS         interface { Name() string; ReadFile(path) ([]byte, error); ReadDir(path) ([]FSFileInfo, error); Exists(path) bool; Watch(path, onChange) error }
type Cache      interface { Name() string; Get/Set/Delete/Invalidate/Clear ... }
```

## Plugin registration

Each plugin package registers itself via `init()` using `core.Register*` functions.
The `Registry` is auto-populated from registered defaults when fields are nil.

```go
// engine/goja/register.go
func init() { core.RegisterEngine(func() core.JSEngine { return &Engine{} }) }

// renderer/react/register.go
func init() { core.RegisterRenderer(func() core.Renderer { return New() }) }
```

## Usage

```go
import pola "github.com/polagonow/pola"
import "github.com/polagonow/pola/core"

app, err := pola.New(&core.Config{
    WebAppPath: "./ui/apps/my-app",
    Registry: &core.Registry{
        // nil fields are filled from init()-registered defaults
    },
})
http.ListenAndServe(":8080", app)
```

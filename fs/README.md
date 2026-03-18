# fs

File system abstraction for the Pola framework.

Each implementation satisfies `core.FS`. The framework reads app source files,
watches for changes in dev mode, and serves static assets — all through this interface.

## Available implementations

| Package | Purpose |
|---------|---------|
| `fs/osfs` | Reads from the OS file system (`os.*`) |
| `fs/embedfs` | Reads from an `embed.FS` (for single-binary deployments) |
| `fs/hybrid` | Tries OS first, falls back to embed (dev vs prod) |

## Usage

```go
import "github.com/polagonow/pola/fs/osfs"

fs := osfs.New("./ui/apps/my-app")
data, err := fs.ReadFile("app/page.tsx")
```

## Embed assets (`POLA_EMBED=true`)

When built with the `embed` tag, use `embedfs` or `hybrid` to serve pre-compiled
assets from the binary without touching the file system at runtime.

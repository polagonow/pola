# fs

File system abstraction for the Pola framework.

Each implementation satisfies `core.FS`. The framework reads app source files,
watches for changes in dev mode, and serves static assets — all through this interface.

## Available implementations

| Package | Purpose |
|---------|---------|
| `fs/osfs` | Reads from the OS file system (`os.*`) |

## Usage

```go
import "github.com/polagonow/pola/fs/osfs"

fs := osfs.New("./ui/apps/my-app")
data, err := fs.ReadFile("app/page.tsx")
```

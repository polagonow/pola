# cache

Page render cache implementations for the Pola framework.

Each implementation satisfies `core.Cache`. The framework caches rendered pages
keyed by URL path. Cache management is exposed via `app.Cache()`.

## Available implementations

| Package | Status | Notes |
|---------|--------|-------|
| `cache/memory` | **FULL** | LRU cache backed by `github.com/hashicorp/golang-lru` |
| `cache/redis` | stub | Redis-backed distributed cache |

## Usage

```go
import "github.com/polagonow/pola/cache/memory"

reg.Cache = memory.New(1000)   // LRU with 1000 entries
```

## Cache management

```go
app.Cache().Clear(ctx)                  // evict all entries
app.Cache().Invalidate(ctx, "/posts/")  // evict all entries with this prefix
app.Cache().Delete(ctx, "/posts/hello") // evict a single entry
```

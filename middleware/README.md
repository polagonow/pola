# middleware

HTTP middleware implementations for the Pola framework.

Each middleware implements `core.Middleware` (wraps an `http.Handler`).
Middleware is applied in order via `Registry.Middleware`.

## Available middleware

| Package | Purpose |
|---------|---------|
| `middleware/logging` | Structured request logging via `core.Logger` |
| `middleware/recovery` | Panic recovery with error logging |
| `middleware/compression` | Gzip/deflate response compression |
| `middleware/cors` | Cross-Origin Resource Sharing, incl. preflight (OPTIONS) handling |
| `middleware/health` | Liveness (`/healthz`) + readiness (`/readyz`, runs registered checks → 200/503) |

## Usage

```go
import (
    "github.com/polagonow/pola/middleware/logging"
    "github.com/polagonow/pola/middleware/recovery"
    "github.com/polagonow/pola/middleware/compression"
)

reg.Middleware = []core.Middleware{
    recovery.New(logger),
    logging.New(logger),
    compression.New(),
}
```

Middleware wraps in order: `recovery → logging → compression → app`.

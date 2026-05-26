# logger

Logger implementations for the Pola framework.

Each implementation satisfies `core.Logger`. The logger is injected into all
framework components via `Registry.Logger`.

## Available implementations

| Package | Notes |
|---------|-------|
| `logger/slog` | Default — wraps `log/slog` |
| `logger/noop` | Discards all output |

## Usage

```go
import sloglogger "github.com/polagonow/pola/logger/slog"

reg.Logger = sloglogger.New()
```

## Interface

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
    Warn(msg string, args ...any)
    With(args ...any) Logger   // returns a logger with key-value context
}
```

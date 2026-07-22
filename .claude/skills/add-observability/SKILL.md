---
name: add-observability
description: Add a logger, metrics, or tracing backend to the Pola framework. Use when asked to integrate a logging library (zap, zerolog), a metrics system (StatsD, OpenTelemetry metrics, Datadog), or a tracing backend alongside the existing slog logger, prometheus/noop metrics, and otel/noop tracers.
---

Observability has three independent extension points, all defined in
`core/interfaces.go` and all registered the same way: a `Plugin()` that calls
`core.ProvideValue[IFACE](r, impl)`. Existing implementations: `logger/slog/`,
`observability/metrics/{prometheus,noop}/`, `observability/tracing/{otel,noop}/`.

## Files to create

| Backend | Files |
|---------|-------|
| Logger | `logger/<name>/<name>.go` + `logger/<name>/plugin.go` |
| Metrics | `observability/metrics/<name>/<name>.go` + `plugin.go` |
| Tracer | `observability/tracing/<name>/<name>.go` + `plugin.go` |

---

## Logger — `core.Logger`

Reference: `logger/slog/slog.go`

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
    Warn(msg string, args ...any)
    With(args ...any) Logger   // child logger with attached key/value pairs
}
```

`args` are alternating key/value pairs (slog-style). Components that want the
logger implement `core.LogAware` and receive it after wiring — your backend only
needs to satisfy the interface.

```go
// logger/<name>/plugin.go
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "logger:<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.Logger](r, New())
        },
    }
}
```

## Metrics — `core.Metrics`

Reference: `observability/metrics/prometheus/prometheus.go` (real) and
`observability/metrics/noop/noop.go` (minimal skeleton).

```go
type Metrics interface {
    Name() string
    Path() string                       // HTTP mount path, e.g. "/_pola/metrics"
    RecordRequest(route, method string, statusCode int, duration time.Duration)
    RecordRender(route string, duration time.Duration)
    Handler() http.Handler              // served at Path()
}
```

The pipeline calls `RecordRequest` for every HTTP request and `RecordRender` for
every page render, and mounts `Handler()` at `Path()`. A push-based backend
(StatsD, Datadog) can return a 404/health handler from `Handler()` and export in
`Record*` instead.

```go
// observability/metrics/<name>/plugin.go — reference: prometheus/plugin.go
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.Metrics](r, New())
        },
    }
}
```

## Tracer — `core.Tracer`

Reference: `observability/tracing/otel/otel.go`.

```go
type Tracer interface {
    Name() string
    StartSpan(ctx context.Context, name string) (context.Context, core.Span)
}

// core.Span:
//   End()
//   SetAttribute(key string, value any)
```

The returned context must carry the span so nested `StartSpan` calls form a
trace tree. Wrap your SDK's span in a small adapter type implementing
`core.Span`.

## Enabling a backend

Apps (and the test combos in `test/combo/`) opt in explicitly:

```go
builder.Use(myname.Plugin())
```

The mage dev targets gate metrics/pprof behind `POLA_METRICS` / `POLA_PPROF`
env vars (see `magefile.go`); the noop implementations are the defaults when
nothing is registered.

## Verify

```
go build ./...
go test ./logger/... ./observability/...
```

For metrics: run an app with the plugin registered and curl the endpoint
(`curl localhost:3000/_pola/metrics`). For tracing: assert spans nest by starting
two spans from the same context in a unit test.

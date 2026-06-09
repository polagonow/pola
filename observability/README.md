# observability

Metrics, tracing, and profiling for the Pola framework.

## Sub-packages

### Metrics (`observability/metrics/`)

| Package | Interface | Notes |
|---------|-----------|-------|
| `metrics/noop` | `core.Metrics` | Default — no-op |
| `metrics/prometheus` | `core.Metrics` | Prometheus `/_pola/metrics` endpoint |

Tracks: request latency, route hits, render duration, bundle build time.

```go
import "github.com/polagonow/pola/observability/metrics/prometheus"

reg.Metrics = prometheus.New()   // mounts /_pola/metrics automatically
```

Enable via env: `POLA_METRICS=true`

### Tracing (`observability/tracing/`)

| Package | Interface | Notes |
|---------|-----------|-------|
| `tracing/noop` | `core.Tracer` | Default — uses `go.opentelemetry.io/otel/trace/noop` |
| `tracing/otel` | `core.Tracer` | OpenTelemetry tracer |

Spans: per-request, per-render, per-JS-call, per-bundle-build.

```go
import "github.com/polagonow/pola/observability/tracing/otel"

reg.Tracer = otel.New()
```

Enable via env: `POLA_TRACING=true`

### Pprof (`observability/pprof/`)

Serves Go profiling data at `/_pola/pprof/`. Disabled by default.

```go
import "github.com/polagonow/pola/observability/pprof"

reg.Pprof = pprof.New()   // mounts /_pola/pprof/
```

Enable via env: `POLA_PPROF=true`

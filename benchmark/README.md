# benchmark

Performance benchmarks for all Pola engines, renderers, and bundlers.

## Running

```bash
# All benchmarks with default combo (goja + esbuild + react)
mage benchmark

# Specific engine
POLA_VM=goja mage benchmark

# With memory profiling
go test -bench=. -benchmem ./benchmark/...
```

## Suites

| File | What it measures |
|---|---|
| `engine_bench_test.go` | JS runtime eval throughput per engine (Goja, V8, Node, QJS) |
| `renderer_bench_test.go` | RSC render throughput (full pipeline) |
| `bundler_bench_test.go` | esbuild incremental vs cold build times |

## Interpreting results

- **ns/op** — nanoseconds per operation; lower is better
- **B/op** — bytes allocated per operation; lower is better
- **allocs/op** — heap allocations per operation; lower is better

Full pipeline benchmarks (`renderer`, `bundler`) require a built app and are skipped
in CI unless `POLA_BENCH=1` is set. Run them locally with `mage benchmark`.

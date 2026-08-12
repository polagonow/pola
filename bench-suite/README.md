# Pola benchmark suite

Reproducible benchmarks comparing **Pola** (every JS engine it ships) against a
**Node.js RSC baseline** (`nodejs-rsc`, `react-server-dom-webpack` — the same
Flight protocol Pola uses). Both sides are RSC, so the comparison isolates the
**runtime**: native Node V8 vs Pola's Go-embedded JS VM. Every number is measured
on the machine that runs it — nothing is estimated. **No winner is declared**;
read `FAIRNESS.md` for where the comparison is and isn't apples-to-apples.

> This directory is `bench-suite/` — deliberately distinct from the repo's Go
> `benchmark/` package. `BENCHMARK/` collides with it on a case-insensitive
> filesystem, and `benchmarks/` differs by only a trailing "s".

## Quick start

```bash
cd bench-suite
pnpm install            # harness deps (autocannon)
pnpm bench              # build + measure every entry, then regenerate RESULTS.md
```

Faster iterations:

```bash
pnpm bench:nodejs-rsc                 # just the RSC baseline
node bench.mjs --only nodejs-rsc --quick  # 3 runs, 3s load (dev sanity)
node bench.mjs --scenarios 1,2     # subset of scenarios
node bench.mjs --skip-install --skip-build   # reuse prior artifacts
node report.mjs                    # regenerate RESULTS.md from results/summary.json
```

`pnpm bench` reproduces everything from clean: for each entry it installs,
cold-builds, warm-builds, starts the server, measures, and stops it — one entry
at a time, no shared `node_modules`. It then regenerates `RESULTS.md` and the
charts in `CHARTS.md`.

📊 **Charts:** [`CHARTS.md`](CHARTS.md) — throughput, latency, memory, async
correctness, build time, and a nested-Suspense streaming timeline (blue = Pola,
orange = Node baseline). Regenerate standalone with `pnpm charts`.

📖 **Glossary:** [`GLOSSARY.md`](GLOSSARY.md) — every short form used here (RSC,
SSR, Flight, TTFB/TTLB, CoV, RSS, JIT, CGO, WASM, …) with reference links.

## What is measured (per scenario, per entry)

- **TTFB** and **time-to-last-byte** (median / p95 / p99 + coefficient of variation)
- **Flush timeline** — timestamp + size of every response chunk, so out-of-order
  streaming is visible
- **Payload bytes** — raw, gzip, and brotli (computed offline so entries with
  different wire compression stay comparable). RSC entries report the Flight
  payload separately from the shell.
- **Client JS bytes** — framework runtime vs app code (raw split via bundler
  metafile; gzip/brotli on the whole bundle)
- **Cold build / warm build / cold start** to first successful response
- **RSS** at idle and under sustained load; peak during build (indicative)
- **Load** via autocannon (req/s, latency percentiles)

Browser-side metrics (hydration/time-to-interactive, and DOM conformance for RSC
entries whose first response is a shell) use Playwright + CDP — see `FAIRNESS.md`.

## Scenarios

1. Static page, no data — baseline render cost
2. Server component awaiting a 50ms source, Suspense-streamed
3. Interactive client-component island in an otherwise server tree
4. Nested Suspense, 3 boundaries resolving at 20/50/200ms

Scenario 3 (client-component island) is **N/A** for the `nodejs-rsc` baseline —
the "use client" bundler machinery is the framework layer Pola provides that a
raw Node RSC baseline lacks; recorded, not approximated.

## Conformance gate

Before measuring, every implementation of a scenario must produce equivalent
**normalized rendered DOM** (scripts, comments, and framework-specific attributes
stripped). Because every entry is RSC (first response is a shell), conformance is
compared from browser-rendered DOM after network-idle. Failures are excluded with
an explanation (in `RESULTS.md` / `FAIRNESS.md`), never silently included.

## Layout

```
bench-suite/
  bench.mjs             orchestrator (install→build→start→measure→stop)
  report.mjs            results/summary.json → RESULTS.md
  lib/                  measure, sizes, rss, load, proc, stats, conformance, env
  entries/
    nodejs-rsc/         Node.js + react-server-dom-webpack — RSC baseline
    pola-goja/          Pola: goja engine (default) + react (RSDW) renderer
    pola-nativersc/     Pola: goja engine + Go-native Flight renderer
    pola-<engine>/      one per JS engine: sobek, v8go, moderncquickjs, quickjsgo, qjs
  results/              raw per-run JSON (gitignored)
  CAPABILITIES.md       Pola capabilities, source-cited (Phase 0)
  FAIRNESS.md           deviation log
  RESULTS.md            generated results (committed)
```

## Adding an entry

Create `entries/<name>/entry.config.mjs` exporting `{ name, kind, install, build,
start, health, scenarios, flight, clientBundles, ... }`. The orchestrator
discovers it automatically. See `entries/pola-goja/entry.config.mjs` for the
contract (`kind: "rsc"` + `flight` enables two-request measurement).

## Entries

- **nodejs-rsc** — Node.js + `react-server-dom-webpack` — **RSC baseline** (apples-to-apples with Pola; isolates the runtime)
- **pola-goja** — Pola, goja engine (pure-Go interpreter) + react/RSDW renderer
- **pola-nativersc** — Pola, goja engine + Go-native Flight renderer
- **pola-v8go** — Pola, V8/JIT engine (CGO) + react renderer
- **pola-sobek** — Pola, sobek engine (pure-Go goja fork) + react renderer
- **pola-moderncquickjs** — Pola, QuickJS engine (CGO) + react renderer
- **pola-qjs** — Pola, QuickJS (fastschema/qjs, **WASM**) — renders correctly sequentially; crashes under concurrent load
- **pola-quickjsgo** — Pola, QuickJS (quickjs-go, CGO) — renders sequentially (except nested Suspense); unstable under concurrent load

Every Pola JS engine is benched with the same app + react renderer, so only the
engine differs. The `node` engine is **excluded** (it shells out to an external
Node.js binary, contradicting the single-binary premise). Engines whose async
rendering doesn't honor the source delay are flagged `async-not-honored` /
`content-missing` by the correctness gate, not reported as fast — see
`FAIRNESS.md`.

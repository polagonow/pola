# Pola benchmark suite

Reproducible benchmarks comparing **Pola** against a raw **Node.js** baseline —
Node.js as the runtime, rendering with `react-dom/server` (`renderToPipeableStream`),
no framework. Every number here is measured on the machine that runs it — nothing
is estimated. **No winner is declared**; read `FAIRNESS.md` for where the
comparison is and isn't apples-to-apples.

> This directory is `benchmarks/` (not `BENCHMARK/`) because the repo already has
> a Go package `benchmark/` and the filesystem is case-insensitive.

## Quick start

```bash
cd benchmarks
pnpm install            # harness deps (autocannon)
pnpm bench              # build + measure every entry, then regenerate RESULTS.md
```

Faster iterations:

```bash
pnpm bench:control                 # just the control
node bench.mjs --only control --quick   # 3 runs, 3s load (dev sanity)
node bench.mjs --scenarios 1,2     # subset of scenarios
node bench.mjs --skip-install --skip-build   # reuse prior artifacts
node report.mjs                    # regenerate RESULTS.md from results/summary.json
```

`pnpm bench` reproduces everything from clean: for each entry it installs,
cold-builds, warm-builds, starts the server, measures, and stops it — one entry
at a time, no shared `node_modules`.

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
4. Nested Suspense, 3 boundaries resolving at 20/50/200ms — **RSC-only**
   (non-RSC entries are recorded **N/A**, not approximated)

Scenarios 1–3 have plain-SSR equivalents for the Node.js control.

## Conformance gate

Before measuring, every implementation of a scenario must produce equivalent
**normalized rendered DOM** (scripts, comments, and framework-specific attributes
stripped). SSR entries are compared from first-response HTML; RSC entries are
compared from browser-rendered DOM after network-idle. Failures are excluded with
an explanation (in `RESULTS.md` / `FAIRNESS.md`), never silently included.

## Layout

```
benchmarks/
  bench.mjs             orchestrator (install→build→start→measure→stop)
  report.mjs            results/summary.json → RESULTS.md
  lib/                  measure, sizes, rss, load, proc, stats, conformance, env
  entries/
    control/            Node.js + raw react-dom/server (the runtime baseline) — built
    pola-default/       Pola: goja engine + react (RSDW) renderer — built
    ...                 pola-v8go, pola-nativersc (Pola variants, added incrementally)
  results/              raw per-run JSON (gitignored)
  CAPABILITIES.md       Pola capabilities, source-cited (Phase 0)
  FAIRNESS.md           deviation log
  RESULTS.md            generated results (committed)
```

## Adding an entry

Create `entries/<name>/entry.config.mjs` exporting `{ name, kind, install, build,
start, health, scenarios, flight, clientBundles, ... }`. The orchestrator
discovers it automatically. See `entries/control/entry.config.mjs` for the
contract (`kind: "rsc"` + `flight` enables two-request measurement).

## Status

Built and measured: **control** (Node.js baseline) and **pola-default**.
Pending: Pola **v8go** and **nativersc** variants. Each is committed once it
builds and passes conformance.

# Fairness log

Every place the comparison deviates from "each framework's documented production
config, untouched" is recorded here. An optimization applied to one entry is
applied to all or to none. Where defaults differ in ways that move a metric, the
difference is **recorded, not normalized away**.

## Ground rules (applied to every entry)

- **Production builds only.** Each entry is built with its framework's documented
  production command and served from the build output, never a dev server.
- **React in production mode.** `NODE_ENV=production` for every React-based entry
  (control sets it at build-time define + runtime env; frameworks set it via
  their production build). Recorded because a dev build would inflate client JS
  and slow rendering.
- **No hand-tuning.** No custom caching, compression, or minifier settings beyond
  what the framework does by default — for any entry, Pola included.
- **One entry at a time.** No concurrent servers, no shared `node_modules`; each
  entry directory installs and builds independently.
- **7 runs / scenario, first 2 discarded** as warmup; median / p95 / p99 + CoV
  reported. Load via autocannon; browser-side via Playwright/CDP (where used).
- **Never estimated.** Build failures, unsupported scenarios, and crashes are
  recorded as those outcomes in `results/` and surfaced in `RESULTS.md`.
- **No winner is declared.**

## Structural differences that move metrics (recorded, not normalized)

### Pola — two-request model (the big one)
Pola's first response for a normal page load is an **empty HTML shell**; the page
content arrives via a **second** request (`Accept: text/x-component`) that streams
the RSC Flight payload, rendered client-side (`renderer/react/react.go:156-177`;
`docs/ssr-caching.md`). Next.js App Router and the control instead put
server-rendered content in the first response.
- **Consequence:** document TTFB/TTLB for Pola measures only the shell; the
  content cost is in the Flight request. The harness therefore measures **both**
  requests for RSC entries and reports them on separate rows.
- **Consequence:** raw first-response HTML is not comparable across entries for
  the conformance gate — Pola's would normalize to near-empty. Conformance for
  RSC entries uses **browser-rendered DOM** (Playwright, network-idle), not raw
  HTML.
- **Consequence:** "hydration duration" has no like-for-like meaning for Pola
  (client render from Flight via `createRoot`, not hydration of server HTML). It
  is redefined per-entry as a browser time-to-interactive mark.

### Pola — default in-memory cache
A default LRU cache exists, but the Flight tee-cache only fills when a route sets
`Route.Revalidate > 0` (`renderer/react/react.go:225-232`). Benchmark scenarios
set no revalidate, so pages re-render every request — comparable to the other
entries' dynamic rendering. Recorded so it isn't mistaken for "Pola has no cache."

### Pola — engine is a pure-Go interpreter
The default `goja` engine has no JIT (`engine/goja/`). This is an architectural
property, not a tuning choice; a V8/JIT variant (`v8go`) is measured separately
so the interpreter-vs-JIT gap is visible rather than hidden.

### Compression on the wire
- **Control:** no gzip/brotli on the wire (raw bytes). The harness computes
  gzip/brotli sizes offline from response bodies so payload size is comparable
  even though transfer encoding differs.
- Other entries: whatever their production default is, recorded per entry. Not
  normalized.

### Scenario 4 (nested RSC Suspense) is RSC-only
Marked **N/A** for the control and for React Router's non-RSC mode rather than
approximated with plain-SSR Suspense.

### Scenario 3 island on non-RSC entries
Plain React has no partial hydration, so the control (and React Router non-RSC)
hydrate the whole document root; the "island" is one interactive component in an
otherwise static tree. RSC entries hydrate only the client component. Recorded
because it changes client JS and hydration cost.

## Harness measurement limitations

- **Build peak RSS** is unreliable for sub-second builds and for bundlers that run
  as a child service process (esbuild) — the single-pid `ps` sampler under-counts.
  Reported as indicative only.
- **RSS** is sampled on the server's **main pid**; a multi-process server would
  under-count. Entries are single-process (Go binary; single Node process) unless
  noted.
- **TTFB/TTLB** are measured over loopback (no network RTT) — they reflect
  server + render cost, not real-world latency.
- **Client JS framework-vs-app split** is RAW output bytes attributed via the
  bundler metafile (post-minify/treeshake); gzip/brotli are reported on the whole
  client bundle (per-module gzip attribution isn't meaningful).

## Per-entry deviations

### control
- Not a framework — raw `react-dom/server` `renderToPipeableStream`. It is the
  floor, not a competitor. No router, no data layer, no client framework runtime
  beyond React itself.
- esbuild (`0.28.2`) is used only to transpile/bundle the control's own
  server/client code; it is not a measured "framework."

### pola-default (goja + react/RSDW renderer)
- Built with `pola build --vm goja --renderer react --bundler esbuild --router
  nextjs --css none --cgo 0` → single static binary; served by running the
  binary. Documented Pola production flow.
- **CSRF + security headers ON** (Pola's `pola new` defaults). GET scenarios are
  unaffected functionally; the shell carries a CSRF `<meta>` and a CSP nonce,
  which add a few bytes to the document (not the Flight payload). Not disabled —
  disabling would be hand-tuning one entry.
- **Cold vs warm build:** Go's compiler cache is process-global and is not
  cleared between cold/warm, so Pola's "cold build" reflects JS bundling +
  linking with a warm Go cache — not a from-scratch Go compile. The control's
  esbuild cold/warm difference is similarly cache-limited. Neither number is a
  from-absolute-zero build; recorded as such.
- **Client JS framework/app split is a filename heuristic** (framework =
  `_client-*` runtime + React `chunks/*` + `ErrorBoundary-*`; app = `Counter-*`
  + route `error-*`/`global-error-*`). The control's split is exact (esbuild
  metafile). This asymmetry is a measurement limitation, not a fairness choice —
  do not compare the two splits to more than ~1 KiB precision.
- **Every page carries an extra framework Suspense boundary** whose fallback is
  `loading.tsx` (Pola wraps pages automatically), in addition to the
  scenario-specific boundaries. The control has only the scenario's boundaries.
  Visible in scenario 2/4 Flight payloads.
- **Two-request model:** the `document` load/timings measure the shell only; the
  `RSC Flight` rows carry the render cost. When comparing "content latency"
  across entries, compare Pola's **Flight** row to the SSR entries' **document**
  row — comparing Pola's document row to an SSR document row compares a shell to
  full content and is meaningless.

## Observed load-tail behavior (recorded, not adjusted)
- **control, scenario 2** (50 connections against a 50ms-suspense SSR stream):
  p50/p95 are healthy (~52/58 ms) but ~24 / 1115 requests exceeded autocannon's
  10 s request timeout, inflating p99/max. This is a real tail characteristic of
  Node `renderToPipeableStream` under concurrent streaming at this connection
  count, not a harness error. Timeouts are surfaced in `RESULTS.md` (⚠ annotation)
  rather than trimmed. Pola scenario 2 showed 0 timeouts at the same concurrency
  but lower peak throughput on fast endpoints (VM-pool serialization).

<!-- Additional entries append their deviations here as they are built. -->

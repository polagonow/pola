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

### Pola — default in-memory cache (and a renderer asymmetry)
A default LRU cache exists (`docs/ssr-caching.md`). The two Pola renderers use it
**differently**, which materially moves the numbers:
- **react** renderer fills the Flight tee-cache **only when `Route.Revalidate > 0`**
  (`renderer/react/react.go:225-232`). With no revalidate set (our scenarios) it
  re-renders every request.
- **nativersc** renderer caches the Flight response **unconditionally**, with
  `TTL: Route.Revalidate` = `0` = *no expiry* (`renderer/nativersc/nativersc.go:184-190`).
  So after the first render, every repeat request to the same URL is served from
  the LRU in microseconds.

This is a real default difference, not a tuning choice. To measure **render cost**
(what the scenarios are about) rather than cache-hit latency, the harness
**cache-busts every measured request** with a unique `?__bench=…` query, applied
**uniformly to every entry** (pages ignore `searchParams`, so rendered output is
byte-identical; the control and react ignore the query and are unaffected). Under
load, autocannon's `idReplacement` puts a unique id in each request's query for
the same effect.

Recorded as a genuine Pola capability, not hidden: **in normal operation Pola's
default LRU serves repeat traffic from memory** (nativersc always; react when a
route opts into `Revalidate`). The benchmark deliberately bypasses that to isolate
render cost. A separate "warm cache / repeat visit" measurement could be added to
show the cached path; it is not part of these render-cost numbers.

### Pola — all JS engines are benched (and several needed wiring)
Every JS engine Pola ships is measured, each with the **same app + react
renderer**, so only the engine differs:

| Engine | Kind | plugin.go shipped? | Wiring added | Async RSC (scenarios 2 & 4) |
|---|---|---|---|---|
| goja | pure-Go interpreter, no JIT | yes | — | ✅ correct |
| sobek | pure-Go (grafana/sobek, goja fork) | **no** | plugin.go + SSRPoolFactory/SSRRuntime bridge | ✅ correct |
| v8go | V8 / JIT (CGO) | **no** | plugin.go + bridge | ✅ correct |
| moderncquickjs | QuickJS (CGO) | yes | — | ✅ correct |
| quickjsgo | QuickJS (CGO) | yes | — | ❌ **broken** (see below) |
| qjs | QuickJS (CGO) | **no** | plugin.go + bridge | ❌ **broken** (see below) |
| node | execs external `node` | no | — | **excluded** |

- **Wiring added (committed framework changes):** `sobek`, `v8go`, and `qjs`
  shipped without a `plugin.go` and implemented an older render contract
  (`StartRender` + `DrainStream(RenderSession)`). Each got a `plugin.go` and a
  `CallRenderFunction`/`DrainStream(StreamHandle)`/`NewSSRPool` bridge mirroring
  goja. No behavior of the already-wired engines changed. Disclosed because it is
  a framework change made to enable benchmarking; the diffs are in this commit
  series.
- **`node` is excluded**, not benched: the `node` engine shells out to an
  external Node.js binary, which contradicts Pola's single-static-binary premise
  and has no `plugin.go`. Recorded as an outcome, not silently dropped.
- **JIT vs interpreter is visible:** goja/sobek are pure-Go interpreters (no
  JIT); v8go is V8 (JIT); the QuickJS bindings are bytecode VMs. Numbers exposed,
  no winner declared.

### Correctness gate for async rendering (why quickjsgo & qjs are flagged)
The harness asserts, per scenario, that the rendered content contains a required
marker **and** — for the async scenarios — that render time clears a floor equal
to the source delay (scenario 2 ≥ 40 ms for a 50 ms source; scenario 4 ≥ 180 ms
for 20+50+200 ms). Engines that return async content *without awaiting* fail this
gate and their scenario-2/4 numbers are recorded as the outcome
`async-not-honored` (or `content-missing` when flaky), **not** reported as fast:
- **quickjsgo** and **qjs** render scenario 2 in ~1–2 ms instead of ~50 ms; `qjs`
  additionally logs `render await: expected JS Promise, got Undefined`. Their
  event-loop/promise handling does not drive the async Server Component +
  Suspense path correctly. Static (scenario 1) and the island (scenario 3) render
  fine; the async scenarios are excluded with this explanation.
- goja, sobek, v8go, moderncquickjs clear the floor (scenario 2 ≈ 50 ms,
  scenario 4 ≈ 270–300 ms) and pass.

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

### pola-nativersc (goja + Go-native Flight renderer)
- Same app source as `pola-default`, built with `--renderer nativersc`. The RSC
  Flight payload is serialized in Go (`renderer/nativersc/flight.go`,
  `reconciler.go`) instead of by react-server-dom-webpack inside the VM.
- **Flight detection differs:** nativersc keys only off `Content-Type:
  text/x-component` (`nativersc.go:106`), while the react renderer also accepts an
  `Accept` header. The harness sends **both** headers (matching the real Pola
  client, which sends `Content-Type`), so this is not a measurement bias.
- **Unconditional Flight caching** (see the renderer-asymmetry note above) — the
  reason the harness cache-busts. Without cache-busting, nativersc scenario 2/4
  would report ~0.16 ms cache-hit latency instead of real render cost.
- Wire format differs from the react renderer (`$L…` lazy refs, `$Sreact.suspense`
  emitted in Go), so Flight payload byte counts are not expected to match
  pola-default to the byte.
- Same client runtime (`@pola/react/client`) as pola-default; client-JS
  framework/app split uses the same filename heuristic.

### pola-v8go (V8/JIT engine + react/RSDW renderer)
- Same app + react renderer as `pola-default`; only the server JS engine differs
  (V8 with JIT instead of the goja interpreter). Isolates the engine's effect.
- **Required a framework patch to work at all** (committed): `engine/v8go` shipped
  without a `plugin.go` and did not implement the current `core.SSRPoolFactory` /
  `core.SSRRuntime` contract (it exposed the older `StartRender` +
  `DrainStream(RenderSession)`). Added `engine/v8go/plugin.go` and a bridge
  (`CallRenderFunction` → `StreamHandle`, `DrainStream(StreamHandle)`,
  `NewSSRPool`) mirroring the goja engine. No behavior of goja/react/nativersc
  was changed. Disclosed here because it's a framework change made to enable one
  entry — the diff is in the same commit series and reproducible.
- **Requires CGO** (`CGO_ENABLED=1`, `--cgo 1`); V8 is statically linked, so the
  binary is ~49 MB vs goja's ~17 MB and is **not** a from-scratch static binary.
  Recorded as a real deployment difference, not normalized.
- Uses the react renderer → does **not** cache Flight by default (Revalidate=0),
  same as pola-default; cache-busting still applied uniformly.
- Observed: on trivial renders V8's per-call CGO boundary cost makes it *slower*
  than goja here; and RSS under sustained load is very high (V8 isolates in the
  VM pool — multiple GB at 50 connections) vs the goja engines' ~130 MiB. Both
  are real datapoints, recorded not adjusted. No winner is declared.

## Observed load-tail behavior (recorded, not adjusted)
- **control, scenario 2** (50 connections against a 50ms-suspense SSR stream):
  p50/p95 are healthy (~52/58 ms) but ~24 / 1115 requests exceeded autocannon's
  10 s request timeout, inflating p99/max. This is a real tail characteristic of
  Node `renderToPipeableStream` under concurrent streaming at this connection
  count, not a harness error. Timeouts are surfaced in `RESULTS.md` (⚠ annotation)
  rather than trimmed. Pola scenario 2 showed 0 timeouts at the same concurrency
  but lower peak throughput on fast endpoints (VM-pool serialization).

<!-- Additional entries append their deviations here as they are built. -->

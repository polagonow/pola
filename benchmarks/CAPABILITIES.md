# Pola Capabilities (for benchmark design)

**Purpose.** This document records what Pola does and how, read from the source
tree at the repo root, so the benchmark comparison is grounded in fact rather
than in assumptions carried over from Next.js. Every claim cites a file path.
Where the source does not determine something, it is marked **`unknown`** and is
*not* inferred from how another framework behaves.

Commit / tree state: branch `paulgrammer/amman`. Paths are repo-relative.

> **Directory note.** The benchmark suite lives at `benchmarks/` (not
> `BENCHMARK/`) because the repo already contains a Go package `benchmark/`
> (`benchmark/engine_bench_test.go`, run via `mage benchmark`) and this
> filesystem is case-insensitive, so `BENCHMARK/` and `benchmark/` are the same
> directory.

---

## 0. TL;DR for the benchmark

- Pola is a Go framework that renders React Server Components inside an embedded
  JS VM and streams them to the browser over React's Flight protocol. Apps
  compile to a single Go binary; no Node.js at runtime (`README.md:8`,
  `CLAUDE.md`).
- **The single most important fact for a fair comparison:** for a normal browser
  page load, Pola's *first* HTTP response is an **empty HTML shell** — the page
  content is **not** server-rendered into that first response. The browser then
  makes a **second** request that streams the RSC Flight payload, which the
  client renders. See [§4](#4-streaming-behavior). This differs fundamentally
  from Next.js App Router (which streams server-rendered HTML in the first
  response). It changes what TTFB, time-to-last-byte, and "hydration" *mean* for
  Pola, and it dictates that the DOM-conformance gate compare *post-render* DOM,
  not raw first-response HTML.

---

## 1. Routing conventions

Pola has **two parallel routing systems** sharing one URL namespace: file-based
frontend pages (RSC) and Go API routes.

### 1a. Frontend pages — file-based, Next.js "app router" style

- The only fully-wired router is `router/nextjs/`; it registers `core.Router`
  via a plugin (`router/nextjs/plugin.go`). `router/std/` and `router/htmx/`
  are stubs per `router/README.md`.
- Pages are discovered from an `app/` directory *inside the web-app directory*.
  The default web-app dir is `web/` (`internal/cli/serve.go`, flag
  `--app-path` default `./web`; documented in `README.md`). The entry generator
  uses `<appDir>/app` as the pages root when that subdir exists
  (`renderer/react/entry.go:75-77`).
- Conventions (`router/nextjs/README.md`; corroborated by
  `examples/blog-e2e-react/web/app/`):
  - `app/page.tsx` → `/`; `app/posts/page.tsx` → `/posts`.
  - Dynamic segment `app/posts/[slug]/page.tsx` → `/posts/:slug`.
  - Catch-all `[...path]`, optional catch-all `[[...path]]`
    (`renderer/react/entry.go:360-371` `stripBrackets`).
  - Route groups `(group)` are stripped from the URL
    (`renderer/react/entry.go:373-382` `isRouteGroup`/`stripParens`).
  - Companion files per segment: `layout`, `error`, `loading`, `not-found`;
    root-level `global-error`, `global-not-found`
    (`renderer/react/entry.go:134-190`; real files in
    `examples/blog-e2e-react/web/app/`).
- A default export is required; the page function receives `{ params,
  searchParams }` (`examples/blog-e2e-react/web/app/page.tsx:84-88`).
- `"use client"` as the first directive marks a client component (bundled and
  loaded separately) — see [§2](#2-rsc-support-and-how-its-wired).
- Page file extensions handled by the React renderer: `.tsx`, `.jsx`, `.ts`,
  `.js` (`renderer/react/react.go:34`).

### 1b. API routes — Go, directory/convention based

- Documented in `routes/README.md`; dispatch/registration code in
  `routes/routes.go`, `routes/pathconv.go`, `routes/reflect.go`. A package under
  `routes/` exports verb methods/functions; the URL derives from the package
  path (`routes/health/` → `/health`).
- **Precedence:** for GET, a matching page wins; non-GET (or GET with no page)
  falls through to the Go API route (`routes/README.md` "Page Priority";
  `router/nextjs/README.md` "Coexistence with API Routes").

**For the benchmark:** all four scenarios are frontend pages under `web/app/`.

---

## 2. RSC support and how it's wired

RSC is first-class. `renderer/react/react.go` package doc: "provides a React
Server Components (RSC) renderer for Pola. It implements streaming SSR via the
RSC Flight wire protocol" (`renderer/react/react.go:1-3`). The renderer reports
`Capabilities() = ["streaming", "rsc"]` (`renderer/react/react.go:36-37`).

**There are two fully-implemented RSC renderers** (both have a `plugin.go`):

1. **`renderer/react/` — the default** (`--renderer react`). The server entry is
   *generated* TypeScript that imports the **real**
   `renderToReadableStream` from `react-server-dom-webpack/server.browser`
   (`renderer/react/entry.go:82`) and defines a global `__render__` that calls
   `renderToReadableStream(React.createElement(Page, props), CLIENT_MANIFEST,
   {...})` (`renderer/react/entry.go:42-52`). Go acquires a VM, invokes
   `__render__`, and drains the resulting JS `ReadableStream`, piping the Flight
   bytes to the HTTP response (`renderer/react/react.go`; VM drain in
   `engine/goja/goja.go`, per exploration). `renderer/react/flight.go` also
   contains a hand-rolled Go Flight encoder (`FlightWriter`, chunk types
   `J`/`I`/`H`/`E`/`S` at `renderer/react/flight.go:31-37`) plus a `WriteRaw`
   path used "when react-server-dom-webpack has already produced the Flight wire
   format — we just pipe the bytes through" (`renderer/react/flight.go:93-100`).
   **Which encoder path is authoritative at runtime is only partially
   determinable from the files read:** the generated entry uses genuine RSDW
   (`entry.go:82`), so the production path is real React Flight piped via
   `WriteRaw`; the native `FlightWriter` chunk encoder appears to be a
   secondary/manual path. Treat the exact division as **partially unknown**.

2. **`renderer/nativersc/`** (`--renderer nativersc`; has
   `renderer/nativersc/plugin.go`). It runs server components in the VM but
   **serializes the Flight wire format natively in Go** by walking the React
   element tree (`renderer/nativersc/flight.go`,
   `renderer/nativersc/reconciler.go`). Its `flight.go` package doc specifies
   the exact wire grammar (rows `<hexId>:<tag?><payload>\n`; `I` import, `E`
   error; reference encodings `$L<hex>` lazy/client, `$@<hex>` promise,
   `$F<hex>` server-reference, etc.). Used by `examples/server-actions-demo`
   (`examples/server-actions-demo/README.md`).

### Server vs client component split

- Detected at bundle time by scanning the file's directive prologue for
  `"use client"` / `'use client'` (`bundler/esbuild/esbuild.go:633`).
- esbuild runs **two passes** (`bundler/esbuild/esbuild.go`): a client bundle
  (browser ESM; each `"use client"` module gets an ESM stub with a client
  reference, `esbuild.go:823`) and a server/pages bundle built with the
  `react-server` export condition (`renderer/react/entry.go:60-62`).
- A **client manifest** (module → `ClientRef{id,name,chunks,async}`,
  `renderer/react/flight.go:16-24`) is produced at build time and injected into
  the server bundle as the esbuild define `__CLIENT_MANIFEST__`
  (`renderer/react/entry.go:17-19`, consumed at `renderer/react/entry.go:46`).
- On the wire, a client component becomes an `I` import row + a lazy reference
  that the browser resolves against the manifest to load the right JS chunk.

### Server actions

- `"use server"` modules are registered with RSDW via `registerServerReference`
  (`renderer/react/entry.go:83-87`) so functions passed as props serialize as
  server references; the client posts invocations to `/_pola/action`
  (`internal/stubpkgs/react/components/Client.tsx`, per exploration;
  `serveraction/` package).

---

## 3. Hydration / client runtime

- **Client entry:** the package specifier `@pola/react/client`
  (`renderer/react/react.go:568-571` `ClientEntry`), backed by
  `internal/stubpkgs/react/components/Client.tsx`.
- **Mount uses `createRoot`, not `hydrateRoot`.** Per the RSC-client
  exploration, `Client.tsx` fetches the Flight stream and parses it with
  `createFromFetch` (Flight parser vendored at
  `internal/stubpkgs/react/components/flight/`), then mounts the resulting tree
  into `#__POLA_ROOT__` (`RootElementID = "__POLA_ROOT__"`,
  `renderer/react/react.go:80-81`) via `createRoot(...).render(...)`. In other
  words the initial tree comes from **parsing the Flight stream on the client**,
  not from hydrating server-emitted HTML.
  - Caveat: `docs/ssr-caching.md:80` mentions "`render`/`hydrateRoot`" in prose.
    The renderer never emits page-content HTML to hydrate (see [§4](#4-streaming-behavior)),
    so the effective model is a client render from Flight. The precise
    `createRoot`-vs-`hydrateRoot` call should be re-confirmed by reading
    `internal/stubpkgs/react/components/Client.tsx` directly during Phase 1;
    marked **`partially unknown`** here.
- **Client bundle:** produced by esbuild pass 1; the client-entry URL is
  surfaced to the HTML shell as `params.ClientScript`
  (`renderer/react/react.go:241-243`).

**Benchmark implication:** because there is no server-rendered content DOM to
hydrate, the task's "hydration duration" metric has no like-for-like equivalent
for Pola. Per the agreed methodology it is redefined per-entry as a
browser-measured *time-to-interactive* mark, and the difference is logged as a
caveat.

---

## 4. Streaming behavior

**Real HTTP streaming, flush-per-chunk.** The Flight response is written through
a `streamWriter` that calls `http.Flusher.Flush()` after each write
(`renderer/react/react.go:379-397`). A request is treated as a Flight request
when `Content-Type` **or** `Accept` equals `text/x-component`
(`ContentType = "text/x-component"`, `renderer/react/react.go:71`; matched at
`renderer/react/react.go:138-139`). Flight responses set `Cache-Control:
no-store` (`renderer/react/react.go:184`).

**Out-of-order Suspense streaming is supported.** In the default react renderer,
each Suspense boundary is fanned out as a goroutine and results are streamed as
they resolve — "first resolved = first flushed"
(`renderer/react/suspense.go:56-91`, comment at line 83). The Flight writer is
concurrency-safe and documented as "one per Suspense boundary"
(`renderer/react/flight.go:39-47`); a placeholder is written immediately while
real content streams later (`renderer/react/flight.go:84-91`
`WriteSuspensePlaceholder`). The `nativersc` renderer implements the same
shape via a deferred-drain phase over pending async subtrees
(`renderer/nativersc/reconciler.go`, per exploration).

### 4a. The two-request model (defining caveat)

For a normal browser GET (no `text/x-component`), Pola serves **only the HTML
shell** and lets the client fetch the content:

- Code comment: *"Pre-render only for not-found pages (the client won't make a
  Flight request for them). Regular pages serve the shell immediately and let
  the client fetch data via a streaming Flight request."*
  (`renderer/react/react.go:156-158`). The HTML path then calls `serveHTML` with
  `ssrData == nil` for regular pages (`renderer/react/react.go:159-177`).
- `serveHTML` emits a document shell containing the client script and (only if
  present) an inline `__POLA_SSR_DATA__` global — no page-content HTML
  (`renderer/react/react.go:235-284`).
- The authoritative doc `docs/ssr-caching.md:9-17` diagrams it:
  - **Request 1** → HTML shell with `<script type="module">` loading the client.
  - **Request 2** → Flight data (`Content-Type: text/x-component`), parsed via
    `createFromFetch()`.

### 4b. Caching (a default difference to record, not normalize)

- A default in-memory LRU cache is always available (`memory.MustNew(0)`,
  1024-entry, `docs/ssr-caching.md:53-55`).
- On a **subsequent** visit to a route whose cache is warm, the shell embeds the
  cached Flight bytes as `__POLA_SSR_DATA__`, and the client renders from the
  embedded string with **no second request** (single-request model,
  `docs/ssr-caching.md:21-30`; embedding at `renderer/react/react.go:267-274`).
- **However**, the Flight tee-cache only fills when the route opts in via
  `Route.Revalidate > 0` (`renderer/react/react.go:225-232`; not-found
  pre-render cache at `renderer/react/react.go:159-169`). So **default dynamic
  pages (no `Revalidate`) re-render on every request** — which is the
  fair-to-measure default. This default-cache behavior will be recorded in
  `FAIRNESS.md`.
- Sets `Vary: Content-Type` so the browser HTTP cache doesn't serve Flight data
  where HTML is expected (`docs/ssr-caching.md:67-69`).

### 4c. Transport specifics

- Streaming is a plain flushed `text/x-component` body over Go's `net/http`;
  chunked transfer encoding is left to `net/http` (no explicit
  `Transfer-Encoding` handling in the files read). No evidence of HTTP/2 server
  push or SSE. Exact on-the-wire transfer headers: **`unknown`** (delegated to
  the Go stdlib).

---

## 5. Supported runtimes

Runtime combos are selected at build time; defaults are overridable via
`POLA_VM` / `POLA_BUNDLER` / `POLA_RENDERER` / `POLA_ROUTER` (`CLAUDE.md`;
`core/env/env.go`). **Defaults: `goja` / `esbuild` / `react` / `nextjs`**
(`internal/cli/build.go` flag defaults; `README.md` CLI tables).

Fully-wired components have a `Plugin()`/`plugin.go`:

| Layer     | Fully-wired (has `plugin.go`)                 | Present but not wired / stub |
|-----------|-----------------------------------------------|------------------------------|
| Engine/VM | `goja` (default), `moderncquickjs`, `quickjsgo` | `sobek`, `v8go`, `qjs` — substantial source, **no `plugin.go`**; `node` execs a `node` binary |
| Bundler   | `esbuild` (default)                            | `vite`, `rollup` — `vite` returns `ErrNotImplemented` (`bundler/vite/vite.go`) |
| Renderer  | `react` (default), `nativersc`                | `vue`, `svelte`, `htmx`, `angular`, `templ`, `mdx` (mdx used via react) |
| Router    | `nextjs` (default)                            | `std`, `htmx` |

Notes:
- `goja` is a **pure-Go interpreter with no JIT** (`engine/goja/`; default per
  `CLAUDE.md`). This is a relevant performance confound versus V8-backed
  competitors and is recorded, not normalized.
- `engine/v8go` (V8, JIT) and `engine/sobek`/`engine/qjs` contain substantial
  implementations (hundreds of lines each) but **no `plugin.go`**, so whether
  they are selectable without additional wiring is **`not cleanly
  determinable`** from source; the `engine/README.md` "stub" labels don't fully
  match the source. For the benchmark, `v8go` is expected to need explicit
  plugin wiring and a CGO build (`--cgo 1`), and that will be verified (or
  recorded as a build-failure outcome) in Phase 1.
- Engine dependencies present in `go.mod`: `github.com/dop251/goja`,
  `github.com/dop251/goja_nodejs`, `github.com/grafana/sobek`,
  `github.com/buke/quickjs-go`, `github.com/fastschema/qjs`.

**Benchmark decision:** Pola is measured as three configs — (a) default
`goja + react`, (b) `v8go` engine variant, (c) `nativersc` renderer variant —
with any that can't be built cleanly recorded as a build-failure outcome.

---

## 6. Production build & start commands

- **Build:** `pola build` runs two stages (`internal/cli/build.go`,
  `README.md` "`pola build`"):
  1. *Bundle* — runs the app with `POLA_BUILD_ONLY=true` to emit JS/CSS assets
     into `./public`.
  2. *Compile* — `go build` with `-trimpath -ldflags "-s -w"` into a single
     binary with the assets embedded.
  - Default output `./bin/<app-name>` (override `-o`). Flags include
    `--vm/--renderer/--bundler/--router/--css`, `--cgo` (`CGO_ENABLED`),
    `--embed-migrations`, `--migrate` (`README.md` "`pola build`").
  - `CGO_ENABLED=0 pola build --vm goja` yields a fully static binary; V8 needs
    `--cgo 1 --vm v8go` (`README.md` examples).
- **Start (production):** there is **no `pola start`**. `serve` is only an alias
  of `pola dev`. Production serving = **run the compiled binary directly**; it
  listens on `PORT` (default `3000`) with host `POLA_ADDRESS`
  (`pola.go` `Addr()`; generated `internal/generators/app/_templates/main_go.tmpl`
  does `pola.Ready()` then `http.ListenAndServe(pola.Addr(), nil)`;
  `examples/blog-e2e-react/Dockerfile` does `pola build ... -o /app` then
  `CMD ["/app"]`).
- **Dev (not for measurement):** `pola dev` (alias `serve`) — hot reload,
  default port 3000, sets `POLA_ENV=development` (`internal/cli/serve.go`,
  `README.md` "`pola dev`").
- The root `magefile.go`/`mage.go` build the **framework itself**, not user
  apps.

**Benchmark decision:** measure the production binary from `pola build`
(default `CGO_ENABLED=0` for goja/nativersc; `--cgo 1` for the v8go variant),
started by executing the binary with a fixed `PORT`.

---

## 7. Minimal app config

- Config file: `Polafile.hcl`, a single `pola { ... }` block; **all fields
  optional** (`polafile/polafile.go`; getter defaults e.g. `WebFramework() →
  "std"`). Env overrides layer on top (`polafile/env.go`). Precedence documented
  as CLI flag → env var → `Polafile.hcl` → hardcoded default (`README.md`).
- A minimal **frontend** app needs, in practice: `package`, `renderer`,
  `engine`, `bundler`, `router`, `app` (e.g. `"web"`), plus the `routes`/
  `actions` dir fields; everything else defaults. Concrete reference:
  `examples/blog-e2e-react/Polafile.hcl` (renderer=react, engine=goja,
  bundler=esbuild, router=nextjs, css=tailwind, app=web, + cache/database
  blocks).
- Scaffolded by `pola new` (`internal/cli/new.go`); default flags there are
  `--renderer react --bundler esbuild --router nextjs --vm goja` (written as
  `engine`), `--css none`, CSRF + security headers on. A scaffolded full app
  ships `main.go`, `go.mod`, `Polafile.hcl`, `routes/health/route.go`, and a
  `web/` React app (`app/layout.tsx`, `app/page.tsx`, `globals.css`, ...).
- Smallest RSC example in-repo: `examples/server-actions-demo`. Smallest overall
  (API-only, no `web/`): `examples/my-api`, `examples/env-config-demo`.

**Benchmark plan:** scaffold each scenario as its own minimal Pola app via
`pola new` + hand-authored `web/app/**` pages, with `csrf`/`security_headers`
left at their documented defaults (and recorded in `FAIRNESS.md`).

---

## 8. Versions (for pinning)

- **Go:** `1.26.5` (`go.mod:3`).
- **React:** `react` `19.2.4`, `react-dom` `19.2.4`,
  `react-server-dom-webpack` `19.3.0-canary-5e9eedb5-20260312` — consistent
  across `examples/*/web/package.json` (e.g.
  `examples/blog-e2e-react/web/package.json`). React is **not vendored** by the
  framework; the internal stub `internal/stubpkgs/react/package.json` declares
  `react`/`react-dom` as `"*"` peer deps, and the real versions are installed
  per-app via pnpm.
- **esbuild (Go):** `github.com/evanw/esbuild v0.20.2` (`go.mod:18`).
- Fumadocs (docs example only): pinned to `16.11.5`
  (`examples/fumadocs-docs/README.md`).

---

## 9. Open items to confirm in Phase 1 (before measuring)

1. Read `internal/stubpkgs/react/components/Client.tsx` directly to confirm
   `createRoot` vs `hydrateRoot` and the exact interactive/hydration mark
   available for the time-to-interactive metric ([§3](#3-hydration--client-runtime)).
2. Empirically confirm the two-request model: build + run one example (e.g.
   `examples/server-actions-demo`), then `curl` a page **without** and **with**
   `-H 'Accept: text/x-component'` to observe shell-vs-Flight ([§4](#4-streaming-behavior)).
3. Confirm whether `--vm v8go` is selectable as-is or needs an `add-vm`-style
   `plugin.go`, and whether the CGO build succeeds on the bench machine
   ([§5](#5-supported-runtimes)).
4. Confirm `--renderer nativersc` builds and serves the same scenario apps
   ([§2](#2-rsc-support-and-how-its-wired)).

---

*Marked `unknown` / `partially unknown` above (not inferred):* exact on-the-wire
transfer-encoding headers ([§4c](#4c-transport-specifics)); the precise runtime
division between the RSDW Flight path and the native `FlightWriter` encoder in
the default renderer ([§2](#2-rsc-support-and-how-its-wired)); the exact
`createRoot`/`hydrateRoot` client call ([§3](#3-hydration--client-runtime)); the
selectability/completeness of the non-default engines
([§5](#5-supported-runtimes)).

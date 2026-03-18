# NEW_ARCHTICITURE_PLAN.md — Implementation Plan (Repo-accurate)

This document converts `NEW_ARCHTICITURE_PLAN_PLAIN.md` into an **implementation plan that matches this repository’s current structure** and build-tag model.

It is written to be executed incrementally (small PRs), with a working mainline at all times.

---

## Goals (what “done” means)

- **Core is framework-agnostic**: no renderer/router/bundler/VM-specific logic in `framework/` beyond interfaces + orchestration.
- **All implementations are plugins**: bundlers, renderers, VMs, routers, CSS, cache. Core uses only interfaces.
- **Build tags remain first-class**: binaries include only implementations selected via build tags (`react`, `esbuild`, `goja`, `v8go`, `embed`, …).
- **Test cross-product stays intact**: existing fixture-based VM×(bundler+renderer) combo testing continues to work and expands to new plugin types.

Non-goals (for MVP):

- Full Next.js feature parity.
- Advanced hot module replacement (HMR). We can do “live reload” first, then graduate to HMR.

---

## Repository mapping (plain doc → this repo)

The plain doc suggests a new `/engine` tree. In this repo, we’ll **achieve the same architecture by evolving existing packages**:

- **Core types + interfaces** → `framework/` (new subpackages are fine if needed, but keep the public API stable).
- **Bundler plugins** → `bundler/*` (already build-tag isolated).
- **Renderer plugins** → `render/*` (already build-tag isolated).
- **JS engine plugins** → `vm/*` (already build-tag isolated).
- **Orchestrator / pipeline** → `framework/` (or `framework/internal/` if unexported).
- **Routers** → `framework/router/*` or `render/react/discovery/*` (eventually migrate “nextjs discovery” into a Router plugin).
- **Manifest / asset contracts** → `bundler/manifest` (already exists).
- **UI tooling** → `ui/` (keep as-is; only integrate via contracts and build outputs).

---

## Architectural contracts (what to implement first)

### Plugin interfaces (core)

Add a minimal, stable set of interfaces and request/response structs. Keep them **narrow** and **data-oriented**.

Plugin categories:

- **Renderer**: SSR/HTML (and optionally streaming) given route match + data + assets + CSS.
- **Router**: resolve a request into a route match, then load data for that match.
- **Bundler**: produce JS/CSS assets and a manifest from entrypoints.
- **CSS**: transform CSS inputs into CSS outputs (or no-op).
- **Cache**: page/fragment cache with TTL and optional tagging.
- **VM**: JS runtime creation + evaluation + module execution hooks as needed by renderer.

Repo placement:

- New types/interfaces under `framework/` (e.g. `framework/plugin/`, `framework/contracts/`, or directly in `framework/`).
- Avoid importing implementation packages (`render/`, `bundler/`, `vm/`) from `framework/`.

### Plugin registry (core)

Implement a registry that can:

- Register implementations by name/type.
- Select “active” implementations from `framework.Config`.
- Validate required plugins are present (clear errors when build tags exclude needed plugins).

Design constraints:

- **No direct coupling**: registry stores interfaces only.
- **Build tags**: implementations register themselves in their own packages behind tags.
- **Embeddable**: allow `embed` builds to exclude bundler code and still run by using prebuilt assets.

### Orchestrator / request pipeline (core)

Implement the request pipeline described in the plain doc, but aligned to this repo’s concepts:

1. Router resolves request → match
2. Cache lookup (page)
3. Router loads data → props/data map
4. Bundler provides build assets/manifest (or “embed assets” provider)
5. CSS processes (optional)
6. Renderer renders HTML/stream
7. Cache store
8. Respond

Keep orchestration in `framework/` and route/bundler/renderer/vm logic in their respective packages.

---

## Phase plan (ordered, shippable milestones)

### Phase 0 — Baseline and naming alignment (1–2 PRs)

- **Deliverable**: a stable directory/namespace plan and no behavioral changes.
- **Work**:
  - Introduce new core packages/filenames for contracts without breaking existing APIs.
  - Add “adapter” shims where needed so existing code continues to compile.
  - Identify the current “entry” for request handling (likely in `framework/framework.go` + renderer/bundler glue) and document it in-code (minimal comments) and in `TASKS.md` if needed.

Acceptance:

- `go test ./...` still passes under the dev tags used today (e.g. `-tags "goja esbuild react"`).

### Phase 1 — Core contracts + registry (MVP spine) (2–4 PRs)

- **Deliverable**: core interfaces + registry with at least one implementation wired through it.
- **Work**:
  - Define request/response structs: `HTTPRequest`, `HTTPResponse`, `RouteMatch`, `RenderRequest`, `RenderResult`, `BuildRequest`, `BuildResult`, `CSSRequest`, `CSSResult`, `CacheOptions`, capabilities enum (optional).
  - Implement `Registry` with typed registration and lookup.
  - Implement `framework.Config` selection logic:
    - Choose plugin names (renderer/router/bundler/css/cache/vm).
    - Validate at startup; return actionable errors when missing due to build tags.
  - Add a “Null”/no-op CSS plugin and a simple in-memory cache plugin (core or plugin package, but must implement the interface).

Acceptance:

- A minimal server path can resolve a request and call a renderer through the registry (even if router is trivial initially).

### Phase 2 — Orchestrator pipeline with adapters (first end-to-end) (2–5 PRs)

- **Deliverable**: the orchestrator exists and can serve HTML using at least one router + one renderer + one asset provider.
- **Work**:
  - Implement `Engine` (or similar) that composes selected plugins and exposes `Handle(ctx, req)`.
  - Implement a **SimpleRouter** plugin (path-based routing) to unblock pipeline development.
  - Implement a **BasicRenderer** plugin to unblock (can be templ or “static HTML” renderer).
  - Provide an **Assets/Bundler abstraction** compatible with `bundler/manifest`:
    - In dev: use `bundler/esbuild` plugin.
    - In embed/release: use an “embedded assets” provider (no bundler code).
  - Integrate caching:
    - Cache key strategy (method + path + vary headers).
    - TTL initially fixed or configurable.

Acceptance:

- E2E test serves a page and asserts HTML output is correct (golden or snapshot).
- Embed-tag build can run without bundler code present.

### Phase 3 — Wrap existing implementations as plugins (migration, not rewrite) (several PRs)

This is the “make it real” phase: adopt the registry/pipeline without losing current features.

#### 3A. Bundler plugin: esbuild

- **Goal**: `bundler/esbuild` implements the `BundlerPlugin` interface.
- **Work**:
  - Create an adapter around the existing esbuild bundler code to produce `BuildResult` (including manifest).
  - Ensure watch mode hooks are possible (even if not used yet).

#### 3B. VM plugin: goja (and others later)

- **Goal**: `vm/goja` provides a `JSEnginePlugin` and `JSRuntime` wrapper.
- **Work**:
  - Provide lifecycle (`Dispose`) and safe per-request runtime creation or pooling (configurable).
  - Define the minimal “module execution contract” needed by the React renderer.

#### 3C. Renderer plugin: React SSR

- **Goal**: `render/react` becomes a `RendererPlugin`.
- **Work**:
  - Move Next.js entry discovery glue behind a Router plugin boundary.
  - Make renderer accept `RenderRequest` (data + route match + assets + CSS + headers).
  - Keep build tags (`react`) selecting the renderer implementation.

#### 3D. Router plugin: Next.js-style router

- **Goal**: convert current Next.js discovery (`render/react/discovery/nextjs/*`) into a `RouterPlugin`.
- **Work**:
  - Router resolves a request into a `RouteMatch` (route id, params, layout chain, entry module).
  - Router loads data (initially: call server-side loader exported from the route module; later: support nested layouts).

Acceptance (Phase 3):

- Existing “Next.js style” app rendering works through the orchestrator, using the registry-selected plugins.

### Phase 4 — CSS pipeline (no-op → Tailwind/Sass) (optional, incremental)

- **Deliverable**: CSSPlugin interface is exercised; at least a passthrough implementation exists.
- **Work**:
  - `CSSPlugin` passthrough (reads CSS, returns unchanged).
  - Tailwind plugin (JIT) behind build tags or external binary integration.
  - Sass plugin (optional).

Acceptance:

- CSS output is included in rendered HTML and/or emitted assets with a manifest entry.

### Phase 5 — Dev mode: watch + live reload (then HMR) (optional, later)

Start with **live reload** (full page reload). Treat “hot reloading” as a capability layered on top.

Plan:

- Define optional “watch” interfaces (do not pollute the base interfaces):
  - `Watch(ctx, onEvent)` for bundler/CSS/router (file discovery) where appropriate.
  - A small dev-server websocket endpoint (e.g. `/_dev/reload`) owned by core, not by plugins.
- For first iteration:
  - On change: rebuild assets + clear caches + notify browser → full reload.

Acceptance:

- Edit route code → rebuild → browser reloads automatically.

### Phase 6 — Advanced caching + streaming SSR (after stability)

- Fragment cache capability in renderer (cache partial HTML), backed by CachePlugin.
- Streaming SSR capability (if React renderer supports it), gated by renderer capabilities.
- Redis cache plugin behind build tags/config.

Acceptance:

- Streaming path works under load and doesn’t break non-streaming path.

---

## Migration strategy (keep mainline green)

- **Prefer adapters over rewrites**: wrap the existing `bundler/esbuild`, `render/react`, and `vm/*` code behind new interfaces first; refactor internals later.
- **Add new path, then switch default**:
  - Implement orchestrator in parallel.
  - Add a config flag to opt in.
  - Once stable, make it default and keep legacy path behind a temporary flag (delete later).
- **Build tags are your safety rail**:
  - Keep registry wiring per implementation in tag-gated files (mirrors existing pattern).

---

## Testing plan (repo-native)

### Contract tests (fast, unit)

For each plugin interface:

- Validate “must” behaviors (e.g. cache TTL, bundler returns manifest, router match invariants).
- Use fake implementations to test orchestrator without requiring real JS execution.

### E2E tests (cross-product)

Use the existing `test/` infrastructure:

- Add a fixture app that exercises: routing → data load → bundling assets → render.
- Ensure it runs across VM×(bundler+renderer) combos automatically (existing `fixture.ForEachApp` pattern).

### Golden/snapshot tests

- Golden HTML for a small set of pages.
- Manifest snapshot for expected entries/chunks.

---

## Risk list and mitigations

- **Risk: interface churn slows progress**
  - Mitigation: keep base interfaces minimal; add optional capability interfaces (e.g. streaming/watch) instead of expanding the base.

- **Risk: embed builds break when bundler is absent**
  - Mitigation: treat “AssetsProvider” as a first-class plugin (bundler in dev, embedded provider in release).

- **Risk: Next.js router responsibilities leak into renderer**
  - Mitigation: router owns filesystem scanning + route tree + match + loader invocation; renderer only renders given match+data.

- **Risk: VM lifecycle/perf issues**
  - Mitigation: define runtime pooling strategy later; start with per-request runtime for correctness, then optimize with pooling + reset hooks.

---

## Concrete work breakdown (file-level touchpoints)

This section is intentionally specific so execution doesn’t stall.

- **Core contracts + registry**
  - `framework/framework.go`: evolve to construct `Engine` from `Config` and registry.
  - `framework/…`: add `contracts` (types) + `plugins` (interfaces) + `registry` + `engine/orchestrator`.

- **Bundler**
  - `bundler/esbuild/bundler.go`: adapt to `BundlerPlugin` and return `BuildResult` compatible with `bundler/manifest`.
  - `bundler/manifest/manifest.go`: ensure it’s the single “asset manifest contract”.

- **React renderer + Next router**
  - `render/react/discovery/nextjs/register.go`: becomes Router plugin registration (tag-gated).
  - `render/react/discovery/nextjs/entry.go`: route match / entry contract, moved behind Router plugin boundary.

- **VM**
  - `vm/*`: expose `JSEnginePlugin` implementations via build tags (goja first).

- **UI**
  - `ui/packages/react/package.json`: ensure build outputs and entrypoints align with bundler expectations (no core imports).

---

## Definition of Done (MVP)

- A running server using:
  - Memory cache plugin
  - Simple router plugin
  - Esbuild bundler plugin (dev)
  - No-op CSS plugin
  - Basic renderer plugin (or React renderer if ready)
- Orchestrator is the only request path.
- E2E test proves pipeline works.
- Build tags produce:
  - Dev binary: `-tags "goja esbuild react"`
  - Release binary: `-tags "embed <vm> react"` (bundler excluded, embedded assets used)


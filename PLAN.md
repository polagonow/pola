# Pola — New Framework Architecture

## Context

The current codebase (`go-react-ssr-v2`) is a working React SSR framework but has testability and
maintainability issues:
- Global `init()` registry creates implicit coupling — hard to unit test
- No FS abstraction — code hits `os.*`/`embed.FS` directly; impossible to mock
- `BridgeConfig` is a flat map, not composable or injectable via DI
- Single renderer — no path to Templ, HTMX, Vue, Svelte, Angular
- No cache layer, no formal middleware, no observability, no CSS pipeline

Goal: full rewrite as **`github.com/polagonow/pola`** — modular, plugin-driven, testable,
minimal-build-friendly (build tags), idiomatic Go naming.

---

## Module

```
module github.com/polagonow/pola
```

---

## Flat Directory Structure (no `plugins/` folder)

**File naming**: each package's main file is named after the package (e.g. `goja/goja.go`,
`react/react.go`, `esbuild/esbuild.go`), not `plugin.go`.

```
pola/
  core/
    types.go            ← shared types — zero external deps
    interfaces.go       ← all interface definitions (no "Plugin" suffix)
    registry.go         ← Registry struct + Config + App
    errors.go
    globals.go          ← well-known JS global names (__REQUEST__, __DEPENDENCY_INJECTION__, etc.)
    env/
      env.go            ← POLA_* env var struct (caarlos0/env)

  internal/
    orchestrator.go     ← App.ServeHTTP — routes req → plugins → response
    pipeline.go         ← build pipeline: scan → bundle → VM setup
    hotreload.go        ← dev hot reload; watches only Renderer.FileExtensions() via FSPlugin.Watch()

  engine/               ← JS runtime engines (was vm/)
    goja/
      goja.go           ← FULL (default)
    v8go/
      v8go.go
    sobek/
      sobek.go
    qjs/
      qjs.go            ← merged qjs variants
    node/
      node.go           ← NEW: node binary via exec
    polyfill/           ← polyfill subscription registry
    eventloop/

  renderer/
    react/              ← FULL (migrated from render/react/)
      shell/
      entry/            ← server entry generation
    templ/              ← STUB
    htmx/               ← STUB
    vue/                ← STUB
    svelte/             ← STUB
    angular/            ← STUB

  router/               ← was "discovery" — renamed, decoupled from renderers
    nextjs/             ← FULL (extension-agnostic file-system routing)
    std/                ← STUB
    htmx/               ← STUB

  bundler/
    esbuild/            ← FULL
    vite/               ← STUB
    rollup/             ← STUB

  fs/                   ← file system abstraction (was framework/assets/)
    osfs/
    embedfs/
    hybrid/

  cache/
    memory/
      memory.go         ← FULL — backed by github.com/hashicorp/golang-lru
    redis/
      redis.go          ← STUB

  injection/            ← Go→JS runtime DI bridge (replaces BridgeConfig)
    do/                 ← samber/do adapter

  css/
    tailwind/           ← FULL — runs tailwindcss CLI, watch + prod modes
    sass/               ← STUB

  middleware/
    logging/
    recovery/
    compression/

  logger/               ← Logger interface + implementations
    slog/               ← default (log/slog)
    noop/

  observability/
    metrics/
      prometheus/       ← FULL — /metrics endpoint
      noop/
    tracing/
      otel/             ← FULL — OpenTelemetry
      noop/             ← default (go.opentelemetry.io/otel/trace/noop)
    pprof/              ← toggleable /debug/pprof/

  static/
    filesystem/

  benchmark/            ← Go benchmark tests for all engines, renderers, bundlers

  cmd/
    server/
      main.go           ← example blog server

  test/
    e2e/
      suite/            ← test suites parameterized by renderer/engine (not React-specific)
        rendering_test.go       ← entry; runs suites against all registered combos
        html_shell_suite.go
        flight_protocol_suite.go
        error_boundary_suite.go
        client_bundle_suite.go
        concurrent_rendering_suite.go
        not_found_suite.go
        layout_composition_suite.go
      renderer/
        react/          ← React-specific e2e assertions
        vue/            ← Vue-specific (skipped until Vue renderer is full)
      engine/           ← engine-specific tests
      router/           ← router tests
      bundler/          ← bundler tests
    combo/              ← registers available engine+bundler+renderer combos
    vm/                 ← registers available VM fixtures
    fixture/

  ui/                   ← pnpm monorepo — ui/packages/jsi/ renamed to @pola/di

  go.mod
  go.sum
  magefile.go           ← updated with new targets + env vars
  lefthook.yml
  README.md
```

---

## Idiomatic Go Naming (no "Plugin" suffix)

All interfaces drop the `Plugin` suffix. Constructor is always `package.New(...)`.

```go
// core/interfaces.go

type Renderer interface { ... }         // react.New(), templ.New(), vue.New()
type Router interface { ... }           // nextjs.New(), std.New()
type Bundler interface { ... }          // esbuild.New()
type JSEngine interface { ... }         // goja.New(), v8go.New(), node.New()
type JSRuntime interface { ... }
type FS interface { ... }               // osfs.New(), embedfs.New(), hybrid.New()
type Cache interface { ... }            // memory.New(), redis.New()
type RuntimeInjector interface { ... }  // do.New()
type CSS interface { ... }              // tailwind.New(), sass.New()
type Middleware interface { ... }       // logging.New(), recovery.New()
type Logger interface { ... }           // slog.New(), noop.New()
type Metrics interface { ... }          // prometheus.New(), noop.New()
type Tracer interface { ... }           // otel.New(), noop.New()
type Pprof interface { ... }            // pprof.New()
type PolyfillRegistry interface { ... }
type HotReloader interface { ... }
```

---

## Core Interfaces (`core/interfaces.go`)

```go
type JSEngine interface {
    Name() string
    NewRuntime(ctx context.Context) (JSRuntime, error)
    RequiredPolyfills() []PolyfillID   // engine declares what it needs
}

type JSRuntime interface {
    Eval(script string) (any, error)
    Call(fn string, args ...any) (any, error)
    Set(name string, value any) error
    Dispose()
}

type Renderer interface {
    Name() string
    Render(ctx context.Context, req RenderRequest) (RenderResult, error)
    Capabilities() []Capability
}

// Router: scans files + resolves requests; not coupled to any renderer
type Router interface {
    Name() string
    // ScanRoutes finds routes; extensions is configurable (e.g. [".tsx", ".vue", ".svelte"])
    ScanRoutes(ctx context.Context, fs FS, appDir string, extensions []string) ([]Route, error)
    Resolve(ctx context.Context, req HTTPRequest) (*RouteMatch, error)
}

type Bundler interface {
    Name() string
    Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
    Watch(ctx context.Context, req BuildRequest, onChange func(BuildResult)) error
}

type FS interface {
    Name() string
    ReadFile(path string) ([]byte, error)
    ReadDir(path string) ([]FSFileInfo, error)
    Exists(path string) bool
    Watch(path string, onChange func(string)) error
}

type Cache interface {
    Name() string
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, val []byte, opts CacheOptions) error
    Delete(ctx context.Context, key string) error
    // Invalidate removes all entries matching a key prefix
    Invalidate(ctx context.Context, prefix string) error
    // Clear removes all entries
    Clear(ctx context.Context) error
}

// App exposes cache management:
// app.Cache().Clear(ctx)
// app.Cache().Invalidate(ctx, "/posts/")
// app.Cache().Delete(ctx, "/posts/hello")

// RuntimeInjector: exposes Go services to JS runtime as __DEPENDENCY_INJECTION__ functions
// User registers DB repos, services in do.Injector → they become JS-callable
type RuntimeInjector interface {
    Name() string
    Inject(ctx context.Context, runtime JSRuntime) error
    Capabilities() []InjectionCapability
}

type CSS interface {
    Name() string
    Process(ctx context.Context, req CSSRequest) (*CSSResult, error)
    Watch(ctx context.Context, req CSSRequest, onChange func(CSSResult)) error
}

type Middleware interface {
    Name() string
    Wrap(next http.Handler) http.Handler
}

type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
    Warn(msg string, args ...any)
    With(args ...any) Logger
}

type Metrics interface {
    Name() string
    RecordRequest(route, method string, status int, d time.Duration)
    Handler() http.Handler    // serves /metrics
}

type Tracer interface {
    Name() string
    StartSpan(ctx context.Context, name string) (context.Context, Span)
}

type Pprof interface {
    Name() string
    Handler() http.Handler   // serves /debug/pprof/
}
```

---

## Registry & App (`core/registry.go`)

```go
type Registry struct {
    Renderer    Renderer
    Router      Router
    Bundler     Bundler
    Engine      JSEngine
    FS          FS
    Cache       Cache
    CSS         CSS
    Logger      Logger          // propagated to all components
    Polyfills   PolyfillRegistry
    Metrics     Metrics         // default: noop
    Tracer      Tracer          // default: noop
    Pprof       Pprof           // default: nil (disabled)
    Middleware  []Middleware
    Injectors   []RuntimeInjector
    // RouteExtensions NOT here — each renderer declares its own via FileExtensions()
}

type Config struct {
    WebAppPath   string
    Dev      bool
    Registry *Registry
}

// App implements http.Handler — drop into any Go HTTP server
type App struct { ... }

func New(cfg *Config) (*App, error)
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (a *App) Build(ctx context.Context) (*BuildArtifacts, error)
```

Usage:
```go
app, err := pola.New(cfg)
http.ListenAndServe(":8080", app)   // App is a plain http.Handler
```

---

## Minimal Build via Build Tags

Same pattern as the current magefile — each implementation is gated by a build tag.
Only the selected engine/bundler/renderer/router/css are compiled into the binary.

```go
// engine/goja/register.go
//go:build goja
package goja
func init() { core.RegisterEngine(New()) }

// renderer/react/register.go
//go:build react
package react
func init() { core.RegisterRenderer(New()) }

// bundler/esbuild/register.go
//go:build esbuild
package esbuild
func init() { core.RegisterBundler(New()) }

// css/tailwind/register.go
//go:build tailwind
package tailwind
func init() { core.RegisterCSS(New()) }
```

The `Registry` auto-populates from `init()` registrations when fields are nil, OR
the user can supply explicit instances (for testing, or when they want multiple).

---

## DI: sambar/do — Business Logic → JS Runtime

The `injection/do` package bridges Go services registered in a `do.Injector` into the JS runtime
as `__DEPENDENCY_INJECTION__` async functions. This is how server pages call Go code:

```go
// cmd/server/main.go
injector := do.New()

// Infrastructure plugins
do.Provide(injector, func(i *do.Injector) (core.FS, error) {
    return osfs.New("./ui/apps/blog-e2e-react"), nil
})
do.Provide(injector, func(i *do.Injector) (core.JSEngine, error) {
    return goja.New(), nil
})

// Domain services — automatically exposed to JS as __DEPENDENCY_INJECTION__ functions
do.Provide(injector, func(i *do.Injector) (*ProjectRepository, error) {
    db := do.MustInvoke[*sql.DB](i)
    return NewProjectRepository(db), nil
})
do.Provide(injector, func(i *do.Injector) (*UserService, error) {
    return NewUserService(do.MustInvoke[*DB](i)), nil
})

// do.RuntimeInjector resolves services and wires their methods into __DEPENDENCY_INJECTION__
app, err := pola.New(&core.Config{
    WebAppPath: "./ui/apps/blog-e2e-react",
    Dev:    true,
    Registry: &core.Registry{
        Injectors: []core.RuntimeInjector{doinjection.New(injector)},
    },
})
http.ListenAndServe(":8080", app)
```

In a React server page:
```tsx
// app/(work)/projects/page.tsx
// __DEPENDENCY_INJECTION__ global is typed via @pola/di (replaces @pola/jsi)
export default async function ProjectsPage() {
    const projects = await __DEPENDENCY_INJECTION__.getProjects()   // → ProjectRepository.GetProjects()
    const user = await __DEPENDENCY_INJECTION__.getUser(id)         // → UserService.GetUser(id)
    return <ProjectList projects={projects} />
}
```

Frontend package `@pola/di` (`ui/packages/di/` — replaces `ui/packages/jsi/`):

Users import the `di` object, never use `__DEPENDENCY_INJECTION__` directly:
```ts
// app/(work)/projects/page.tsx
import di from '@pola/di'

export default async function ProjectsPage() {
    const projects = await di.getProjects()
    const user = await di.getUser(id)
    return <ProjectList projects={projects} />
}
```

The `@pola/di` package is split into:
- **Static base** (`packages/di/src/index.ts`) — thin re-export of the runtime global:
  ```ts
  const di = __DEPENDENCY_INJECTION__
  export default di
  ```
- **Auto-generated types** (`packages/di/src/generated.d.ts`) — produced by Pola at build time
  by inspecting the DI graph and emitting typed declarations:
  ```ts
  // AUTO-GENERATED by pola — do not edit
  declare const __DEPENDENCY_INJECTION__: {
      getUser: (id: string) => Promise<{ id: string; name: string }>
      getProjects: () => Promise<Array<{ id: string; title: string }>>
  }
  export default __DEPENDENCY_INJECTION__
  ```
  Generation step runs as part of `pipeline.Build()` before the bundler runs.

---

## Environment Configuration (`core/env/`)

All Pola env vars are prefixed `POLA_`. Parsed via `github.com/caarlos0/env/v6`.

```go
// core/env/env.go
package env

import "github.com/caarlos0/env/v6"

type Env struct {
    VM          string `env:"POLA_VM"           envDefault:"goja"`
    Bundler     string `env:"POLA_BUNDLER"      envDefault:"esbuild"`
    Renderer    string `env:"POLA_RENDERER"     envDefault:"react"`
    Router      string `env:"POLA_ROUTER"       envDefault:"nextjs"`
    CSS         string `env:"POLA_CSS"          envDefault:"none"`
    WebAppPath   string `env:"POLA_WEBAPP_PATH"  envDefault:"./app"`   // was WebAppPath
    Dev          bool   `env:"POLA_DEV"          envDefault:"false"`
    EmbedAssets  bool   `env:"POLA_EMBED"        envDefault:"true"`

    MetricsEnabled bool   `env:"POLA_METRICS"       envDefault:"false"`
    MetricsPath    string `env:"POLA_METRICS_PATH"  envDefault:"/metrics"`

    PprofEnabled   bool   `env:"POLA_PPROF"         envDefault:"false"`
    PprofPath      string `env:"POLA_PPROF_PATH"    envDefault:"/debug/pprof"`

    TracingEnabled bool   `env:"POLA_TRACING"       envDefault:"false"`
}

func Load() (*Env, error) {
    e := new(Env)
    if err := env.Parse(e); err != nil {
        return nil, err
    }
    return e, nil
}
```

Magefile env vars renamed to match: `POLA_VM`, `POLA_BUNDLER`, `POLA_RENDERER`, etc.
(backward-compat aliases `JS_VM`, `JS_BUNDLER` kept for now but deprecated in README)

---

## Public Env Vars (`POLA_PUBLIC_*`)

Env vars matching `POLA_PUBLIC_*` are automatically exposed to client components
(same pattern as Next.js `NEXT_PUBLIC_*`).

Mechanism:
- At build time, `bundler/esbuild/` reads all `POLA_PUBLIC_*` env vars
- Passes them to esbuild's `define` map as `process.env.POLA_PUBLIC_FOO = "value"`
- Client components use them as normal `process.env` access:

```tsx
// 'use client'
export default function Footer() {
    return <p>Version: {process.env.POLA_PUBLIC_APP_VERSION}</p>
}
```

Server components have access to ALL env vars (no restriction) since they run in Go.

---

## Router (was "Discovery")

Renamed from `Discoverer`/`discovery` to `Router` throughout. Key changes:
- Standalone package, not part of any renderer
- File extensions come from the registered `Renderer.FileExtensions()` — not from config
- Renderers do not define their own routers; they only declare which file extensions they handle

```go
// core/interfaces.go — Renderer declares its own extensions
type Renderer interface {
    Name() string
    FileExtensions() []string   // e.g. react returns [".tsx", ".jsx"], vue returns [".vue"]
    Render(ctx context.Context, req RenderRequest) (RenderResult, error)
    Capabilities() []Capability
}

// core/interfaces.go — Router asks renderer for extensions at scan time
type Router interface {
    Name() string
    // ScanRoutes uses exts provided by the pipeline (sourced from renderer.FileExtensions())
    ScanRoutes(ctx context.Context, fs FS, appDir string, exts []string) ([]Route, error)
    Resolve(ctx context.Context, req HTTPRequest) (*RouteMatch, error)
}

// internal/pipeline.go — wires renderer extensions into router scan
func (p *Pipeline) ScanRoutes(ctx context.Context) ([]Route, error) {
    exts := p.registry.Renderer.FileExtensions()
    return p.registry.Router.ScanRoutes(ctx, p.registry.FS, p.cfg.WebAppPath, exts)
}
```

Renderer extension declarations:
```go
// renderer/react/plugin.go
func (r *React) FileExtensions() []string { return []string{".tsx", ".jsx"} }
// renderer/vue/plugin.go
func (v *Vue) FileExtensions() []string { return []string{".vue"} }
// renderer/svelte/plugin.go
func (s *Svelte) FileExtensions() []string { return []string{".svelte"} }
// renderer/templ/plugin.go
func (t *Templ) FileExtensions() []string { return []string{".templ"} }
```

---

## Polyfill Subscription Model

Each engine declares what polyfills it needs. Pipeline injects only those.

```go
// engine/goja/plugin.go
func (g *Goja) RequiredPolyfills() []core.PolyfillID {
    return []core.PolyfillID{
        polyfill.ReadableStream,
        polyfill.MessageChannel,
        polyfill.AbortController,
    }
}
// engine/node/plugin.go
func (n *Node) RequiredPolyfills() []core.PolyfillID {
    return nil  // Node has all these built-in
}
```

Available IDs: `polyfill.Promise`, `polyfill.ReadableStream`, `polyfill.MessageChannel`,
`polyfill.AbortController`, `polyfill.MicrotaskQueue`

---

## Node.js Engine

```go
// engine/node/plugin.go
// Runs server bundle via node binary (for users who don't need single-binary)
// Communication: stdin/stdout JSON protocol for __DEPENDENCY_INJECTION__ calls
type Node struct { Bin string }   // Bin default "node"
```

Execution patterns:
```bash
node --eval "require('./bundle.js').__render__('Page', '{}')"
node -e "process.stdout.write(chunk)"   # streaming via stdout
```

---

## Cache — LRU via golang-lru

`cache/memory/memory.go` uses `github.com/hashicorp/golang-lru` (no manual eviction logic):

```go
// cache/memory/memory.go
import lru "github.com/hashicorp/golang-lru"

type Memory struct {
    lru *lru.Cache
}

func New(size int) *Memory {
    c, _ := lru.New(size)
    return &Memory{lru: c}
}
```

`app.Cache()` accessor exposes `Clear` and `Invalidate` for runtime cache management.

---

## Embed Assets (`POLA_EMBED=true`)

When `POLA_EMBED=true`, the bundler output (JS bundles, CSS) is embedded in the Go binary
using `//go:embed`. The `fs/embedfs/embedfs.go` plugin serves them from `embed.FS`.
Build tag `embed` triggers the embed path in the same pattern as the current codebase.

---

## TailwindCSS Plugin (FULL implementation)

```go
// css/tailwind/plugin.go
// Runs `tailwindcss` CLI (standalone binary or npx)
// Dev mode: watch + incremental rebuild
// Prod mode: full purge + minify
type Tailwind struct {
    Bin        string   // "tailwindcss" or path to standalone binary
    ConfigPath string   // tailwind.config.js (optional for v4)
    InputPath  string
    OutputPath string
}
```

Build tag: `//go:build tailwind`
Env var: `JS_CSS=tailwind`

---

## Observability

### Logger
- `logger/logger.go`: `Logger` interface
- `logger/slog/`: wraps `log/slog` (default)
- `logger/noop/`: no-op
- Injected into all packages at build time via `Registry.Logger`

### Metrics
- Default: noop (`observability/metrics/noop/`)
- Full: `observability/metrics/prometheus/` — `/metrics` endpoint
- Tracks: request latency, route hits, VM pool size, render duration, bundle build time

### Tracing
- Default: noop (`observability/tracing/noop/`) — `go.opentelemetry.io/otel/trace/noop`
- Full: `observability/tracing/otel/` — OTel tracer
- Spans: per-request, per-render, per-JS-call, per-bundle-build

### Pprof
- `observability/pprof/plugin.go`
- Enabled by setting `Registry.Pprof = pprof.New()`
- Mounts at `/debug/pprof/`
- Default: nil (disabled)

---

## Tests — Renderer/VM/Bundler Agnostic

**Unit tests live next to the code they test** (Go convention):
```
engine/goja/goja_test.go
engine/polyfill/registry_test.go
router/nextjs/match_test.go
router/nextjs/scan_test.go
bundler/esbuild/bundler_test.go
fs/osfs/osfs_test.go
cache/memory/memory_test.go
injection/do/do_test.go
css/tailwind/tailwind_test.go
...
```

**E2E test suites** in `test/e2e/` — parameterized over registered combos, not hard-coded to React+Goja:
```
test/e2e/
  rendering_test.go       ← entry; iterates all registered combos
  suite/
    html_shell_suite.go         ← generic HTML checks (any renderer)
    server_rendering_suite.go   ← generic SSR correctness
    concurrent_suite.go
    not_found_suite.go
  renderer/
    react/
      flight_protocol_suite.go  ← RSC-specific (React combos only)
      suspense_suite.go
      layout_suite.go
    vue/                        ← skipped until Vue renderer is FULL
  engine/
    polyfill_suite.go
  router/
    nextjs_suite.go
  bundler/
    client_bundle_suite.go
combo/
  esbuild_react_goja.go   ← registers React+esbuild+goja combo
  esbuild_react_node.go
vm/
  goja.go, v8go.go, ...
fixture/
```

---

## Benchmarks

```
benchmark/
  engine_bench_test.go    ← Goja vs V8 vs Node vs QJS: render throughput, latency
  renderer_bench_test.go  ← RSC render throughput
  bundler_bench_test.go   ← esbuild incremental vs cold build times
  README.md
```

Run: `go test -bench=. -benchmem ./benchmark/...`

---

## Updated Magefile

New env vars added:
```bash
JS_VM=goja          # engine: goja (default), v8go, sobek, qjs, node
JS_BUNDLER=esbuild  # bundler: esbuild (default), vite, rollup
JS_RENDERER=react   # renderer: react (default), templ, htmx, vue, svelte, angular
JS_ROUTER=nextjs    # router: nextjs (default), std
JS_CSS=none         # css: none (default), tailwind, sass
EMBED_ASSETS=1      # embed assets in binary (default: 1)
CGO_ENABLED=1       # CGO for v8go (default: 1)
PPROF=0             # enable pprof endpoint (default: 0)
METRICS=0           # enable prometheus metrics (default: 0)
```

New mage targets:
```go
func Run() error         // dev server — unchanged
func Build() error       // binary — tags include JS_CSS, JS_ROUTER
func Test() error        // unit tests only (fast)
func TestE2E() error     // e2e tests with current VM+bundler+renderer combo
func TestAll() error     // test against all registered combos
func Benchmark() error   // go test -bench=. ./benchmark/...
func Lint() error        // golangci-lint + eslint
func UiLint() error
func UiFormat() error
func UiFormatCheck() error
func InstallHooks() error
func Clean() error
```

`buildTags()` extended to include router and CSS:
```go
func buildTags() string {
    tags := []string{jsVM, jsBundler, jsRenderer, jsRouter}
    if jsCss != "none" { tags = append(tags, jsCss) }
    if embedAssets == "1" { tags = append(tags, "embed") }
    return strings.Join(tags, " ")
}
```

---

## Migration Map

| Old path | New path |
|---|---|
| `framework/contract/` | `core/types.go` |
| `framework/interfaces.go` | `core/interfaces.go` |
| `framework/globals/` | `core/globals.go` |
| `framework/hotreload/` | `internal/hotreload.go` |
| `framework/assets/disk/` | `fs/osfs/` |
| `framework/assets/embed/` | `fs/embedfs/` |
| `framework/framework.go` | `internal/pipeline.go` + `internal/orchestrator.go` |
| `vm/goja/` | `engine/goja/` |
| `vm/v8go/` | `engine/v8go/` |
| `vm/sobek/` | `engine/sobek/` |
| `vm/qjs/` + variants | `engine/qjs/` |
| `vm/polyfill/` | `engine/polyfill/` (redesigned: subscription model) |
| `vm/eventloop/` | `engine/eventloop/` |
| `bundler/esbuild/` | `bundler/esbuild/` |
| `bundler/manifest/` | `bundler/esbuild/` (inlined) |
| `render/react/` | `renderer/react/` |
| `render/react/discovery/nextjs/` | `router/nextjs/` |
| `render/react/shell/` | `renderer/react/shell/` |
| `test/e2e/` | `test/e2e/` (parameterized, not React-specific) |
| `test/fixture/` | `test/fixture/` |
| `example/blog/main.go` | `cmd/server/main.go` |
| `ui/packages/jsi/` (`@pola/jsi`) | `ui/packages/di/` (`@pola/di`) — auto-generated types |

---

## Implementation Phases

### Phase 1 — Core scaffold
- [x] `go.mod` with `github.com/polagonow/pola`
- [x] `core/`: types, interfaces, registry, errors, globals
- [x] `internal/`: orchestrator (ServeHTTP), pipeline, hotreload
- [x] Root `README.md`

### Phase 2 — Logger + FS + Polyfill registry
- [x] `logger/`: interface, slog impl, noop impl
- [x] `fs/osfs/`, `embedfs/`, `hybrid/`
- [x] `engine/polyfill/registry.go` (subscription model)

### Phase 3 — Engine plugins
- [x] `engine/goja/` (with polyfill subscription)
- [x] `engine/v8go/`, `sobek/`, `qjs/`
- [x] `engine/node/` (exec-based)
- [x] `engine/eventloop/`

### Phase 4 — Router + Bundler
- [x] `router/nextjs/` (extension-agnostic)
- [x] `router/std/`, `htmx/` (stubs)
- [x] `bundler/esbuild/`
- [x] `bundler/vite/`, `rollup/` (stubs)

### Phase 5 — Renderers
- [x] `renderer/react/` (full — migrated)
- [x] `renderer/templ/`, `htmx/`, `vue/`, `svelte/`, `angular/` (stubs)

### Phase 6 — DI + Cache + Middleware + CSS
- [x] `injection/do/`
- [x] `cache/memory/`, `redis/` stub
- [x] `middleware/logging/`, `recovery/`, `compression/`
- [x] `css/tailwind/` (FULL), `sass/` stub
- [x] `static/filesystem/`

### Phase 7 — Observability
- [x] `observability/metrics/prometheus/` + noop
- [x] `observability/tracing/otel/` + noop
- [x] `observability/pprof/`

### Phase 8 — Wire + Test + Benchmark
- [x] `cmd/server/main.go` (do DI, all plugins wired)
- [x] Migrate + restructure `test/` (renderer-agnostic suites)
- [x] `benchmark/` package
- [x] Updated `magefile.go`

### Phase 9 — Docs + Skills
- [x] `README.md` in every package
- [x] `.agents/skills/` per package group (update stale skills to new arch)

---

## Verification

1. `go build ./...` — compiles cleanly
2. `go vet ./...` + `golangci-lint run` — clean
3. `mage run` — blog server starts; hot reload works
4. `http.ListenAndServe(":8080", app)` — App is a valid `http.Handler`
5. `mage test` — unit tests pass
6. `mage testE2E` — all e2e suites pass (React+goja+esbuild combo)
7. `go test -bench=. ./benchmark/...` — benchmarks run, results printed
8. `curl localhost:8080/metrics` — Prometheus metrics (when `METRICS=1`)
9. `PPROF=1 mage run` → `/debug/pprof/` accessible
10. Hot reload: edit `.tsx` → browser reloads
11. `JS_VM=node mage run` — works with Node.js engine
12. `JS_CSS=tailwind mage run` — Tailwind CSS processed and served

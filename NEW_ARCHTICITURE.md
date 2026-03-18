# 🌐 1. Project Overview

**Goal:**
Build a **modular, plugin-driven web platform in Go** with support for:

* Renderer plugins (React, Vue, HTMX, Templ)
* Router plugins (Next.js-style, simple path, HTMX)
* Bundler plugins (ESBuild, Rollup, Webpack, Vite)
* CSS plugins (Tailwind, Sass, PostCSS)
* JS engine plugins (Goja, V8)
* Cache plugins (page/fragment/data)
* Hot reload / dev server
* Logger, config, middleware, static assets, i18n
* Optional advanced: streaming SSR, analytics, PWA, database adapters, API routes

**Core Principles:**

1. All plugins must be **decoupled**.
2. Communication only via **interfaces/contracts**.
3. Dev/prod modes must be **clearly separated**.
4. Hot reloading is **optional and isolated**.
5. Engines, bundlers, and renderers are **replaceable**.

---

# 🏗️ 2. Project Structure

```
/engine
  /core
    types.go
    interfaces.go
  /runtime
    js_runtime.go
    engine_manager.go
  /plugins
    /renderer
      react/plugin.go
      vue/plugin.go
      templ/plugin.go
      htmx/plugin.go
    /router
      next/plugin.go
      http/plugin.go
      htmx/plugin.go
    /bundler
      esbuild/plugin.go
      rollup/plugin.go
      vite/plugin.go
      webpack/plugin.go
    /css
      tailwind/plugin.go
      sass/plugin.go
      postcss/plugin.go
    /cache
      memory/plugin.go
      redis/plugin.go
      file/plugin.go
    /js
      goja/plugin.go
      v8/plugin.go
    /logger
      console/plugin.go
      file/plugin.go
    /config
      env/plugin.go
      json/plugin.go
    /middleware
      auth/plugin.go
      logging/plugin.go
      compression/plugin.go
    /static
      filesystem/plugin.go
  /internal
    orchestrator.go
    pipeline.go
    plugin_registry.go
    hot_reload_manager.go
  /cmd
    server/main.go
```

---

# 🧩 3. Core Plugin Interfaces

**Renderer:**

```go
type RendererPlugin interface {
    Name() string
    Render(ctx context.Context, req RenderRequest) (RenderResult, error)
    Capabilities() []Capability
}
```

**Router:**

```go
type RouterPlugin interface {
    Resolve(ctx context.Context, req HTTPRequest) (*RouteMatch, error)
    Load(ctx context.Context, match *RouteMatch) (map[string]any, error)
}
```

**Bundler:**

```go
type BundlerPlugin interface {
    Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
    Watch(ctx context.Context, req BuildRequest, onUpdate func(BuildResult)) error
}
```

**CSS:**

```go
type CSSPlugin interface {
    Process(ctx context.Context, req CSSRequest) (*CSSResult, error)
    Watch(ctx context.Context, req CSSRequest, onUpdate func(CSSResult)) error
}
```

**Cache:**

```go
type CachePlugin interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, val []byte, opts CacheOptions) error
}
```

**JS Engine:**

```go
type JSEnginePlugin interface {
    Name() string
    NewRuntime(ctx context.Context) (JSRuntime, error)
}
```

**JS Runtime:**

```go
type JSRuntime interface {
    Eval(script string) (any, error)
    Call(fn string, args ...any) (any, error)
    Set(name string, value any) error
    Dispose()
}
```

**Hot Reload:**

```go
type HotReloadManager interface {
    Watch(ctx context.Context) error
    NotifyChange(file string) error
    Subscribe(eventType string, callback func())
}
```

---

# 🧪 4. Plugin Dependency Table

| Plugin Type   | Dependencies                           | Notes                                            |
| ------------- | -------------------------------------- | ------------------------------------------------ |
| Renderer      | JS Engine, Bundler, CSS, Cache         | Decides runtime for SSR                          |
| Router        | Cache, optionally Renderer             | Resolves route → renderer                        |
| Bundler       | JS Engine                              | Supports SSR + client builds                     |
| CSS           | None (optional JS engine for Tailwind) | Can chain PostCSS plugins                        |
| JS Engine     | None                                   | Polyfills injected here                          |
| Cache         | None                                   | Supports page, fragment, and data caching        |
| Hot Reload    | Renderer, Bundler, CSS, JS Engine      | Orchestrates dev updates                         |
| Logger        | None                                   | Used globally for all events                     |
| Config        | None                                   | Provides plugin settings                         |
| Middleware    | Router, Renderer                       | Pre/post request hooks                           |
| Static Assets | None                                   | Serves files, integrates with bundler hash paths |

---

# 🚀 5. Build Pipeline / Request Flow

```text
HTTP Request
    ↓
Router.Resolve() → RouteMatch
    ↓
Cache (page-level)
    ↓
Router.Load() → route data
    ↓
Bundler.Build() → JS + SSR assets
    ↓
CSS.Process() → styles
    ↓
Renderer.Render(data + assets + CSS)
    ↓
Cache (store fragments/pages)
    ↓
HotReloadManager (dev mode triggers reloads)
    ↓
HTTP Response
```

---

# 🏁 6. Development Phases

**Phase 1 (MVP)**

* Plugin registry & orchestrator
* Memory cache
* Simple router (path-based)
* Templ renderer
* Goja JS engine
* Dev server with file watcher

**Phase 2 (Core features)**

* ESBuild bundler + Watch
* CSS pipeline (Tailwind + Sass)
* Hot reload manager integration
* Basic middleware (logging/auth)

**Phase 3 (Advanced renderer)**

* Next.js-style router (nested layouts, dynamic routes)
* Fragment caching & tag invalidation
* Multi-renderer support (React/Vue/HTMX)
* SWR caching / incremental rebuilds

**Phase 4 (Production-ready)**

* Redis cache plugin
* V8 JS engine plugin
* Streaming SSR
* Error handling/fallback plugin
* Config/environment plugin
* Static assets plugin

**Phase 5 (Optional/experimental)**

* i18n plugin
* Analytics / telemetry plugin
* Service Worker / PWA plugin
* Database adapters / API route plugin
* Security / sandbox plugin

---

# 🧭 7. Testing Plan

* Unit tests for **each plugin** (interface compliance)
* Integration tests for **pipeline flow**
* Hot reload tests (CSS, JS, renderer updates)
* Cache tests (page, fragment, data, SWR)
* Multi-engine tests (Goja + V8)
* Bundler tests (SSR + client builds)
* Router tests (dynamic routes, layouts)

---

# 🏗️ 8. Dev / Prod Mode Considerations

| Feature    | Dev Mode             | Prod Mode              |
| ---------- | -------------------- | ---------------------- |
| Bundler    | Watch, incremental   | Optimized bundle       |
| CSS        | Watch, no minify     | Minified, purged       |
| Renderer   | Partial updates      | Full SSR               |
| Hot Reload | Enabled              | Disabled               |
| Cache      | Memory, short TTL    | Redis, long TTL, SWR   |
| JS Engine  | Pooled / per-request | Pooled for performance |
| Logger     | Verbose              | Info/Error only        |

---

# ⚡ 9. Golden Rules

1. **All communication via interfaces**
2. **Plugin isolation**: no plugin should directly access another plugin’s internals
3. **Pipeline separation**: rendering, bundling, CSS, JS engines, cache all modular
4. **Hot reload must be optional and non-destructive**
5. **Cache keys deterministic**: route + params + locale + auth
6. **Dev mode ≠ prod mode behavior**
7. **Engine + bundler + renderer compatibility** must be clearly defined

---

# 🧠 10. LLM / Vibe-Code Workflow

1. Generate **core interfaces** first
2. Implement **registry** to manage plugins
3. Implement **orchestrator** to route request → renderer → cache → response
4. Add **core plugins** (MVP phase)
5. Incrementally add **advanced plugins** (Phase 3-5)
6. Integrate **hot reload** after pipeline works
7. Add **tests** for each layer
8. Switch **JS engines** or **bundlers** without touching renderer logic

---

This is a **full system blueprint** that covers every layer and all plugins.

It’s ready to give to an LLM to **vibe code your entire modular platform**, layer by layer.

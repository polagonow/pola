# engine

JavaScript engine implementations for the Pola framework.

Each engine implements `core.JSEngine`. SSR-capable engines also implement
`core.SSRPoolFactory` to create a pool of `core.SSRRuntime` instances from a
compiled server bundle. The engine declares which polyfills it needs via
`RequiredPolyfills()` — the pipeline injects only those.

## Available engines

Status legend:

- **SSR** — implements `NewSSRPool` and ships a `Plugin()`, so it can be selected
  to render Server Components. These are the production engines.
- **Experimental** — has a working VM (via `NewVMPool`) for evaluating JS, but no
  SSR pool and no `Plugin()`, so it cannot yet drive RSC rendering.

| Package | Build tag | Status | Notes |
|---------|-----------|--------|-------|
| `engine/goja` | `goja` | **SSR** | Default engine; pure Go; uses `goja_nodejs` event loop |
| `engine/moderncquickjs` | `moderncquickjs` | **SSR** | Modern C QuickJS (CGO) |
| `engine/quickjsgo` | `quickjsgo` | **SSR** | QuickJS Go bindings |
| `engine/sobek` | `sobek` | Experimental | Fork of goja; VM pool only, no SSR pool |
| `engine/v8go` | `v8go` | Experimental | V8 via CGO; VM pool only, no SSR pool |
| `engine/qjs` | `qjs` | Experimental | QuickJS; VM pool only, no SSR pool |
| `engine/node` | — | Experimental | Spawns the `node` binary via exec (no single-binary, no SSR pool) |

## Sub-packages

| Package | Purpose |
|---------|---------|
| `engine/polyfill` | Polyfill registry + JS sources (MicrotaskQueue, ReadableStream, …) |
| `engine/eventloop` | Event loop utilities shared across engines |

## Build tags

Select the engine at compile time:

```bash
go build -tags goja ./...         # Goja (default)
go build -tags v8go ./...         # V8 via CGO
POLA_VM=goja mage build           # via Magefile env var
```

## Adding a new engine

See `.claude/skills/add-vm/SKILL.md`.

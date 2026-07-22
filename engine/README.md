# engine

JavaScript engine implementations for the Pola framework.

Each engine implements `core.JSEngine`. SSR-capable engines also implement
`core.SSRPoolFactory` to create a pool of `core.SSRRuntime` instances from a
compiled server bundle. The engine declares which polyfills it needs via
`RequiredPolyfills()` — the pipeline injects only those.

## Available engines

| Package | Build tag | Status | Notes |
|---------|-----------|--------|-------|
| `engine/goja` | `goja` | **FULL** | Default engine; pure Go; uses `goja_nodejs` event loop |
| `engine/sobek` | `sobek` | stub | Fork of goja |
| `engine/v8go` | `v8go` | stub | Requires CGO |
| `engine/qjs` | `qjs` | stub | QuickJS |
| `engine/moderncquickjs` | `moderncquickjs` | stub | Modern C QuickJS |
| `engine/quickjsgo` | `quickjsgo` | stub | QuickJS Go bindings |
| `engine/node` | — | FULL | Runs `node` binary via exec; no single-binary |

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

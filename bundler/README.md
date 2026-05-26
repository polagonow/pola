# bundler

JS bundler implementations for the Pola framework.

Each bundler implements `core.Bundler`. The framework calls `Build` at startup
and `Watch` in dev mode. The bundler runs two passes: a client bundle (browser ESM)
and a server bundle (CJS, react-server conditions).

## Available bundlers

| Package | Build tag | Status | Notes |
|---------|-----------|--------|-------|
| `bundler/esbuild` | `esbuild` | **FULL** | Two-pass esbuild pipeline with manifest |
| `bundler/vite` | `vite` | stub | |
| `bundler/rollup` | `rollup` | stub | |

## Sub-packages

| Package | Purpose |
|---------|---------|
| `bundler/manifest` | Client component manifest types |
| `bundler/structs` | Shared bundler struct types |

## Build tags

```bash
go build -tags esbuild ./...      # esbuild (default)
POLA_BUNDLER=esbuild mage build   # via Magefile env var
```

## Adding a new bundler

See `.agents/skills/add-bundler/SKILL.md`.

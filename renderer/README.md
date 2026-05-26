# renderer

UI renderer implementations for the Pola framework.

Each renderer implements `core.Renderer`. It declares which file extensions it handles
(e.g. `.tsx` for React) and renders pages given a `RenderRequest`. Full renderers
also implement `core.BundleLoader` to receive the compiled server bundle from the pipeline.

## Available renderers

| Package | Build tag | Status | Notes |
|---------|-----------|--------|-------|
| `renderer/react` | `react` | **FULL** | React Server Components with Flight protocol |
| `renderer/templ` | `templ` | stub | Go Templ |
| `renderer/htmx` | `htmx` | stub | HTMX |
| `renderer/vue` | `vue` | stub | Vue 3 |
| `renderer/svelte` | `svelte` | stub | Svelte |
| `renderer/angular` | `angular` | stub | Angular |

## React renderer sub-packages

| Package | Purpose |
|---------|---------|
| `renderer/react/shell` | HTML shell rendering (the outer `<html>` document) |

## Build tags

```bash
go build -tags react ./...        # React (default)
POLA_RENDERER=react mage build    # via Magefile env var
```

## Adding a new renderer

See `.agents/skills/add-renderer/SKILL.md`.

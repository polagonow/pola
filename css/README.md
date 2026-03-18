# css

CSS processor implementations for the Pola framework.

Each implementation satisfies `core.CSS`. The framework calls `Process` at startup
and `Watch` in dev mode.

## Available implementations

| Package | Build tag | Status | Notes |
|---------|-----------|--------|-------|
| `css/tailwind` | `tailwind` | **FULL** | Runs `tailwindcss` CLI (standalone or npx) |
| `css/sass` | `sass` | stub | Sass/SCSS |

## Usage

```go
import "github.com/polagonow/pola/css/tailwind"

reg.CSS = tailwind.New()   // uses "tailwindcss" binary on PATH
```

Or with a custom binary:

```go
reg.CSS = &tailwind.Tailwind{
    Bin:        "/path/to/tailwindcss",
    ConfigPath: "./tailwind.config.js",   // optional for v4
}
```

## Build tags

```bash
go build -tags tailwind ./...      # Tailwind
POLA_CSS=tailwind mage build       # via Magefile env var
```

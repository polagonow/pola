---
name: add-css-processor
description: Add a new CSS processor to the Pola framework behind the core.CSS interface. Use when asked to add, integrate, or wire a CSS toolchain such as Less, Stylus, Lightning CSS, UnoCSS, or vanilla-extract alongside the existing tailwind, sass, and postcss processors.
---

A CSS processor implements `core.CSS` (`core/interfaces.go`): it compiles an input
stylesheet to an output file and re-runs on change in dev mode. The bundler receives
it through `core.BundleInput.CSSProcessor` and invokes it during the client bundle
pass. Apps select it with the Polafile `css` attribute / `--css` flag / `POLA_CSS`
(values today: `tailwind`, `sass`, `none`; `postcss` exists as a building block).

## Files to create

| File | Purpose |
|------|---------|
| `css/<name>/<name>.go` | `core.CSS` implementation |
| `css/<name>/plugin.go` | `Plugin() core.Plugin` providing it into the registry |

---

## Step 1 — Implement `core.CSS`

**`css/<name>/<name>.go>`** — reference: `css/tailwind/tailwind.go`

```go
package myname

import (
    "context"

    "github.com/polagonow/pola/core"
)

type Processor struct{}

func New() *Processor { return &Processor{} }

func (p *Processor) Name() string { return "<name>" }

// Process compiles inputPath into outputPath once (production build).
func (p *Processor) Process(ctx context.Context, inputPath, outputPath string) error {
    // run the toolchain; prefer a pure-Go implementation so `pola build`
    // stays Node-free at runtime
    return nil
}

// Watch re-processes on changes and calls onChange after each successful
// rebuild so the dev server can push a browser reload.
func (p *Processor) Watch(ctx context.Context, inputPath, outputPath string, onChange func()) error {
    return nil
}
```

Keep the processor **pure Go** if at all possible (tailwind and sass are) — a
processor that shells out to Node breaks Pola's no-Node-at-runtime story.

## Step 2 — Register via `Plugin()`

**`css/<name>/plugin.go`** — reference: `css/tailwind/plugin.go`

```go
package myname

import "github.com/polagonow/pola/core"

// Plugin returns the <name> CSS processor plugin.
func Plugin() core.Plugin {
    return core.PluginFunc{
        PluginName: "<name>",
        Fn: func(r *core.Registry) {
            core.ProvideValue[core.CSS](r, New())
        },
    }
}
```

## Step 3 — Expose it to the CLI

The CLI maps the `css` value to the plugin import in the generated wiring via
`internal/autoload` (search for `"tailwind"` under `internal/` to find the
mapping) and validates allowed values in `internal/cli/new.go`
(`"CSS processor (tailwind, sass, none)"`). Add `<name>` in both places. The
mage build also treats the CSS choice as a test build tag (`runtimeTags()` in
`magefile.go` appends `POLA_CSS` unless `none`) — no change needed there.

## Verify

```
go build ./...
go test ./css/...
```

Then run a real app on it:

```
pola new css-test -y --css <name>
cd css-test && pola dev
```

The dev server must produce the compiled stylesheet in the bundle output, and
edits to the input CSS must trigger a rebuild + browser reload.

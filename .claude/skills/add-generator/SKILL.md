---
name: add-generator
description: Add a new `pola generate <x>` scaffold generator to the Pola CLI. Use when asked to add, extend, or create a CLI generator/scaffold command (e.g. generate policy, generate job, generate webhook) alongside the existing model/repository/service/dto/action/route/scaffold/zod/seed/docs generators.
---

CLI generators are self-contained packages under `internal/generators/<name>/`
implementing the `generators.Generator` interface, registered via `init()` and a
blank import in `internal/generators/all/register.go`. The generate command tree
(`pola generate <x>`) is built dynamically from the registry, and generators that
also implement the `Destroyer` interface get `pola destroy <x>` support for free.

## Files to create

| File | Purpose |
|------|---------|
| `internal/generators/<name>/<name>.go` | Generator impl + `init() { generators.Register(...) }` |
| `internal/generators/all/register.go` | Add the blank import (one line) |

---

## Step 1 — Implement `generators.Generator`

**`internal/generators/<name>/<name>.go`** — reference: `internal/generators/seed/seed.go`
(minimal, embedded literal template) and `internal/generators/dto/dto.go`
(field-spec parsing + Destroyer).

```go
package myname

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"

    "github.com/polagonow/pola/internal/generators"
    "github.com/polagonow/pola/internal/project"
)

// MyGenerator scaffolds <what it writes>.
type MyGenerator struct{}

func init() { generators.Register(&MyGenerator{}) }

func (g *MyGenerator) Name() string        { return "<name>" }
func (g *MyGenerator) Description() string { return "<one-line description>" }

// AfterHooks run after successful generation (shell or Go hooks).
func (g *MyGenerator) AfterHooks() []generators.Hook {
    return []generators.Hook{generators.CmdHook("gofmt", "-w", ".")}
}

func (g *MyGenerator) Command() *cobra.Command {
    return &cobra.Command{
        Use:     "<name> [Name] [field:type ...]",
        Short:   "<short help>",
        Args:    cobra.MinimumNArgs(1),
        RunE:    g.run,
        Example: `  pola generate <name> Thing name:string`,
    }
}

func (g *MyGenerator) run(cmd *cobra.Command, args []string) error {
    projectDir, err := project.FindRoot()
    if err != nil {
        return err
    }
    filePath := filepath.Join(projectDir, "<dir>", "<file>.go")
    // Respects the persistent --force / --skip-collision-check flags:
    if err := generators.CheckCollision(cmd, filePath); err != nil {
        return err
    }
    if err := os.WriteFile(filePath, src, 0o644); err != nil {
        return fmt.Errorf("write %s: %w", filePath, err)
    }
    fmt.Printf("Created %s\n", filePath)
    return generators.RunAfterHooks(g, projectDir)
}
```

Useful helpers already in the tree:

| Helper | Purpose |
|--------|---------|
| `project.FindRoot()` | Locate the app root (walks up to `Polafile.hcl`) |
| `generators.CheckCollision(cmd, path)` | Honors `--force` / `--skip-collision-check` |
| `generators.RunAfterHooks(g, dir)` / `CmdHook` / `FuncHook` | Post-generation hooks (gofmt, npm install, …) |
| `model.ParseArgs(args)` (`internal/generators/model`) | Parse `Name field:type{opts}:modifier` specs |
| `schema.SnakeCase(name)` | Naming convention for generated files |
| `polafile.Load(dir)` | Read `Polafile.hcl` (ORM, app dir, package manager, …) |

Large generators embed templates (`embed.FS` or string constants) — see
`internal/generators/docs/` for a multi-file template + npm-install after-hook.

## Step 2 — Support `pola destroy` (recommended)

Implement the `Destroyer` interface so `pola destroy <name> …` can reverse the
generator — return the exact paths `run` would write for the same args:

```go
func (g *MyGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
    return []string{filepath.Join(projectDir, "<dir>", "<file>.go")}, nil
}
```

## Step 3 — Register the package

**`internal/generators/all/register.go`** — add one blank import, keeping the
list alphabetical:

```go
_ "github.com/polagonow/pola/internal/generators/<name>"
```

## Verify

```
go build ./...
go run ./cmd/pola generate <name> --help
```

Then scaffold an app and run the generator for real (`pola new gen-test -y`,
`pola generate <name> Thing`, check the emitted file compiles, then
`pola destroy <name> Thing --dry-run` to confirm Artifacts matches). Also
document the new generator in `.claude/skills/pola/references/cli.md`.

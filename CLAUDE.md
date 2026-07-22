# CLAUDE.md

Pola is a Go framework for React Server Components: `.tsx` pages render inside a
Go-embedded JS VM (goja by default), stream to the browser over React's Flight
protocol, and call Go code through a typed bridge. Apps compile to a single static
Go binary — no Node.js at runtime. The `pola` CLI lives in `cmd/pola`.

## Commands

Build automation uses [Mage](https://magefile.org) (`magefile.go`). Run targets with
`mage <target>` or `go run mage.go <target>`.

| Command | Purpose |
|---|---|
| `mage build` | Build the `pola` CLI into `bin/pola` |
| `mage test` | Unit tests with `-race` |
| `mage teste2e` | E2E tests (slow; builds bundles, runs every VM × bundler × renderer combo) |
| `mage testall` | Unit + E2E |
| `mage lint` / `mage fmt` | golangci-lint run / fmt |
| `mage check` | Full gate: lint + vet + race tests |
| `mage benchmark` | Benchmarks in `benchmark/` |

Runtime combos are selected with build tags (defaults: `goja esbuild react nextjs`,
overridable via `POLA_VM` / `POLA_BUNDLER` / `POLA_RENDERER`). A single test run
looks like:

```bash
go test -tags "goja esbuild react nextjs" -run TestName ./path/...
```

## Skills

Project skills live in `.claude/skills/` — Claude Code loads them automatically when
relevant, or invoke one explicitly with `/skill-name`.

### Working on this framework

| Skill | Use for |
|---|---|
| `add-vm` | Add a JS engine (goja, v8go, sobek, quickjs, …) |
| `add-bundler` | Add a bundler (Vite, Rollup, Parcel, …) |
| `add-renderer` | Add a UI renderer (Vue, Svelte, Solid, …) |
| `add-polyfill` | Add a Web API polyfill to the engine layer |
| `add-e2e-test` | Add an E2E suite/case under `test/e2e/suite/` |
| `add-web-framework` | Add an HTTP framework adapter (std, gin, echo, chi, …) |
| `add-middleware` | Add an HTTP middleware under `middleware/` |
| `add-cache` | Add a cache backend (`core.Cache`) |
| `add-storage-driver` | Add a file storage driver (`storage.Storage`) |
| `add-session-store` | Add a session store (gorilla/sessions) |
| `add-mailer-transport` | Add a mail transport or email template renderer |
| `add-css-processor` | Add a CSS processor (`core.CSS`) |
| `add-database-adapter` | Add a GORM/Ent database dialect |
| `add-observability` | Add a logger, metrics, or tracing backend |
| `add-generator` | Add a `pola generate <x>` CLI generator |

### Building apps with Pola

| Skill | Use for |
|---|---|
| `pola` | Scaffold and extend apps with the `pola` CLI (`pola new` / `pola generate` / `pola dev` / `pola build`), Polafile.hcl, migrations, MCP tools |

## Conventions

- Pluggable components (engines, bundlers, renderers, caches, middleware, …) register
  via a `Plugin() core.Plugin` function in their package (`core.PluginFunc` +
  `core.ProvideValue`/`AddMiddleware`) — no init() registries or build tags on
  implementation files (database dialects are the init()-based exception). See the
  matching `add-*` skill before touching those layers.
- Package-level READMEs (`engine/`, `bundler/`, `renderer/`, `test/`) document each
  subsystem and point to the relevant skill.
- Run `mage check` before committing; git hooks are managed by lefthook
  (`mage installhooks`).

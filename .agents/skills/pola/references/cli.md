# Pola CLI reference

Every command, subcommand, and flag, with the exact files each generator writes.

Run the CLI either from an installed binary (`pola …`) or from source
(`go run github.com/polagonow/pola/cmd/pola …`). A prebuilt `bin/pola` exists in the framework
repo. **Global flags** work everywhere: `-v/--verbose`, `--cwd <dir>` (run as if started in `<dir>`),
`-h/--help`, `--version`.

---

## `pola new [app-name]`

Scaffold a new app. The name is optional on the CLI — `pola new` prompts for one. If the name
contains `/` it is treated as a Go module path: the last segment becomes the directory, the full
path becomes the `module` line in `go.mod` (e.g. `github.com/acme/admin` → dir `admin/`,
module `github.com/acme/admin`).

| Flag | Default | Notes |
|------|---------|-------|
| `--renderer` | `react` | View renderer |
| `--bundler` | `esbuild` | JS bundler |
| `--router` | `nextjs` | Router style |
| `--css` | `tailwind` | `tailwind`, `sass`, `none` (auto-adjusted by `--ui`) |
| `--vm` | `goja` | JS engine (written to Polafile as `engine`) |
| `--ui` | `none` | `shadcn`, `mui`, `slds`, `ads`, `carbon`, `patternfly`, `fluentui`, `antd`, `none` |
| `--pm` | auto | `npm`, `pnpm`, `yarn` (auto-detected if unset) |
| `--module` | — | Go module path; auto-derived when the name is a module path |
| `--dependencies` | — | Comma-separated npm version overrides, e.g. `"react@19.2.4,tailwindcss@^4.3.0"` |
| `--csrf` | `true` | Enable CSRF protection |
| `--security-headers` | `true` | Enable security headers |
| `--test-framework` | `vitest` | `vitest`, `jest`, `none` |
| `--skip-tests` | `false` | Skip generating test files / infra |
| `--pola-path` | — | Local path to pola source → adds a `replace` directive (dev against unpublished pola) |

**UI compatibility (validated at runtime):** `--ui=shadcn` requires `--css=tailwind` **and**
`--renderer=react`; `--ui` ∈ {`slds`, `ads`, `carbon`, `fluentui`, `antd`} requires `--renderer=react`.
When `--ui` is set (and `--css` wasn't explicitly passed), the CSS processor is auto-derived from the
UI's deps (e.g. `carbon`/`slds` pull in Sass; UIs without a CSS requirement set `css = none`).

**What it does:** creates the dir and scaffolds renderer/UI templates → writes `Polafile.hcl`
locking the choices and module versions → `go mod tidy` → `<pm> install` in `web/` → stubs
`@pola/actions` and `@pola/react` into `node_modules` → runs the `js:bridge` generator.

```bash
pola new                                          # prompts for name
pola new my-app
pola new github.com/acme/admin                    # dir=admin, module=github.com/acme/admin
pola new my-app --css=tailwind --ui=shadcn
pola new admin --ui=antd --pm=pnpm
pola new local-dev --pola-path=../pola            # develop against a local pola checkout
```

---

## `pola dev` (alias: `serve`)

Run in development mode with hot reload. A Go watcher polls `.go`, `.tmpl`, `go.mod`, `go.sum`,
`Polafile.hcl` and respawns the server on change; a JS watcher re-bundles `.tsx`/`.ts` changes and
pushes a browser reload over WebSocket.

| Flag | Default (env) | Notes |
|------|---------------|-------|
| `-p`, `--port` | `3000` (`PORT`) | Server port |
| `--renderer` | `react` (`POLA_RENDERER`) | |
| `--bundler` | `esbuild` (`POLA_BUNDLER`) | |
| `--router` | `nextjs` (`POLA_ROUTER`) | |
| `--css` | `tailwind` (`POLA_CSS`) | |
| `--vm` | `goja` (`POLA_VM`) | |
| `--app-path` | `./web` (`POLA_WEBAPP_PATH`) | Path to the web app dir |
| `--csrf` | `true` (`POLA_CSRF`) | |
| `--security-headers` | `true` (`POLA_SECURITY_HEADERS`) | |
| `--image-processing` | — (`POLA_IMAGE_PROCESSING`) | `imaging` → enables `/_pola/image` + the `ImageProcessing.processURL` bridge binding |

Defaults resolve: CLI flag → env var → `Polafile.hcl` → hardcoded default.

```bash
pola dev
pola dev --port 8080
PORT=4000 pola dev
pola dev --vm goja --css tailwind
```

---

## `pola build`

Two-stage production build:

1. **Bundle stage** — runs your app with `POLA_BUILD_ONLY=true` to emit JS/CSS into `./public`
   (override dir with `POLA_PUBLIC_DIR`).
2. **Compile stage** — `go build` with embedded assets and `-ldflags="-s -w"`.

| Flag | Default (env) | Notes |
|------|---------------|-------|
| `-o`, `--output` | `./bin/<app-name>` | Output binary path |
| `--renderer` / `--bundler` / `--router` / `--css` / `--vm` | from Polafile/env | Same as `dev` |
| `--cgo` | `1` (`CGO_ENABLED`) | Value of `CGO_ENABLED` passed to `go build` |
| `--app-path` | `./web` (`POLA_WEBAPP_PATH`) | |
| `--csrf` / `--security-headers` | `true` | |
| `--image-processing` | — | `imaging` |

```bash
pola build
pola build -o ./bin/myapp
CGO_ENABLED=0 pola build --vm goja            # fully static binary (goja/sobek only)
pola build --cgo 1 --vm v8go                  # build with V8 (needs CGO + C toolchain)
```

---

## `pola generate` (aliases: `gen`, `g`)

With no subcommand, regenerates the actions bridge and prints an overlay path. With a subcommand,
scaffolds code.

**Common flags inherited by every subcommand:**

| Flag | Default | Notes |
|------|---------|-------|
| `--force` | `false` | Overwrite existing files |
| `--skip-collision-check` | `false` | Skip the file-exists check |
| `--actions-dir` | `./actions` | Path to the actions directory |
| `--ts-out` | — | Path for the generated `.d.ts` |

### `generate action <Name>`
Scaffold an action struct (the bridge surface). Files: `actions/<snake>_action.go` (+ test).
- `--service <Name>` — wire methods to a named service via DI (`NewXAction(svc *services.X)`).
```bash
pola generate action Blog
pola generate action Products --service=Product
```

### `generate js:bridge`
Parse action structs and write the TypeScript declarations consumed by `@pola/actions`. Same as the
implicit step in `pola new`/`pola dev`. No flags.

### `generate model <Name> [field:type …]`
ORM model/schema from field defs. ORM comes from `Polafile.hcl` (`database.orm`). Files:
`db/models/gorm/<snake>.go` (gorm) or `db/models/schema/<snake>.go` (ent).
- `--skip-migration` — don't auto-generate a migration.
```bash
pola generate model User name:string email:string:uniq age:int
pola generate model Article title:string:index body:text author:references
pola generate model Comment body:text commentable:references{polymorphic}
pola generate model User name:string avatar:references{StorageBlob}
```

### `generate repository <Name> [field:type …]` (alias `repo`)
Interface + ORM-specific implementation + shared `repositories/pagination.go`. Files:
`repositories/<snake>_repository.go` and `repositories/gorm/<snake>_repository.go`.

### `generate service <Name> [field:type …]` (alias `svc`)
Service struct depending on the repository interface. File: `services/<snake>_service.go` (+ test).

### `generate route <Name> [methods…]`
HTTP handlers in `routes/<segments>/route.go`. Methods may be space- or comma-separated; default `GET`.
- `--service <Name>` — wire handlers to a service via DI.
```bash
pola generate route Posts
pola generate route Posts GET,POST
pola generate route Posts/Comments GET POST DELETE
pola generate route Posts GET,POST,DELETE --service=Post
```

### `generate page <Name> [field:type …]` (alias `p`)
Renderer-specific CRUD pages using the renderer/UI from `Polafile.hcl`. Files:
`web/app/<plural>/page.tsx` (list), `web/app/<plural>/[id]/page.tsx` (show),
`web/app/<plural>/create/page.tsx`, `web/app/<plural>/[id]/edit/page.tsx`, plus
`web/components/<plural>/{create-form,edit-form,delete-button,list-view,…}.tsx`.

### `generate scaffold <Name> [field:type …]` (alias `s`)
The composite: **model + repository + service + action + route + zod + pages** (+ migration).
Skip individual parts:
`--skip-model`, `--skip-repository`, `--skip-service`, `--skip-action`, `--skip-route`,
`--skip-zod`, `--skip-views`, `--skip-migration`.
```bash
pola generate scaffold Product name:string price:float description:text
pola generate scaffold Product name:string --skip-route
```

### `generate zod <Name> [field:type …]` (alias `z`)
TypeScript Zod schema. File: `web/schemas/<snake>.ts` (under the app dir).

### `generate migration <name>` (alias `mi`)
Diff ORM models against `db/migrations/` and emit a versioned migration via Atlas, into
`database.migrations.directory`.
- `--env <name>` (default `development`) — resolve adapter/dev-url.
- `--dev-url <url>` — override the dev database URL.
```bash
pola generate migration CreateUsers
pola generate migration init --dev-url "sqlite://file?mode=memory"
```

### `generate storage`
Create `StorageBlob` (file metadata) + `StorageAttachment` (polymorphic join) models and configure
the `storage` block in `Polafile.hcl`.
- `--driver` (default `fs`) — `fs` or `rclone`.
- `--root` (default `uploads`) — local dir for `fs`, `remote:path` for `rclone`.
- `--config-path` — rclone config file (rclone driver only).
```bash
pola generate storage
pola generate storage --driver rclone --root myremote:bucket/path --config-path /etc/rclone/rclone.conf
```
Then attach to a model: `pola generate model User avatar:references{StorageBlob}`.

### `generate mailer <Name> [actions…]`
Scaffold a mailer struct + email templates. Each action arg becomes a method on the mailer plus
matching HTML + text templates. At least one action is required. Renderer and app dir come from
`Polafile.hcl`; configure delivery via the `mailer` block. Files: the Go struct at
`mailers/<name>_mailer.go`, and per-action templates under `<app>/mailers/<name>_mailer/`
(e.g. `web/mailers/<name>_mailer/`).
```bash
pola generate mailer User welcome reset_password
pola generate mailer Order confirmation shipped
```

### `generate mcp <sub>`
Model Context Protocol artifacts (see `references/recipes.md`).

| Subcommand | Action |
|------------|--------|
| `init` | Add an `mcp { … }` block to `Polafile.hcl` so the autoload wires the plugin |
| `tool <Name> [--no-di]` | `mcp/tools/<snake>_tool.go`. DI flavor by default (constructor takes `*core.Registry`); `--no-di` emits a simpler `init()`-registered typed tool |
| `resource <Name>` | `mcp/resources/<snake>_resource.go` |
| `prompt <Name>` | `mcp/prompts/<snake>_prompt.go` |

```bash
pola generate mcp init
pola generate mcp tool Greeting
pola generate mcp tool Echo --no-di
pola generate mcp resource AppConfig
pola generate mcp prompt Summarize
```

---

## `pola db <sub>`

Database management. Connection comes from `Polafile.hcl` (`database`) merged with the `--env` block,
or `--url` to override.

> ⚠️ **The migration runner is sqlite-only.** Any other adapter errors with:
> *"adapter `<x>` not yet supported for pola db commands; currently only sqlite is supported"*.
> The Polafile may still declare postgres/mysql for app runtime, but `pola db …` operates on sqlite.

| Subcommand | Purpose | Extra flags |
|------------|---------|-------------|
| `migrate` | Apply pending migrations | `--version <v>` (migrate to a specific version) |
| `rollback` | Roll back applied migrations (uses `-- atlas:down` directives) | `--step <n>` (default 1) |
| `status` | Table of applied vs pending | — |
| `reset` | Drop **all tables**, then re-run every migration (destructive) | — |
| `schema:load` | Apply all migrations to a fresh database | — |

All accept `--env <name>` (default `development`) and `--url <url>`.

```bash
pola db migrate
pola db migrate --version 20240101120000
pola db rollback --step 3
pola db status
pola db reset --url "sqlite:dev.db"
```

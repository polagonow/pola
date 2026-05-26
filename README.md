# Pola

A Go framework for **React Server Components (RSC)** — implements the Flight streaming protocol, Next.js-style file conventions, and a pluggable multi-VM architecture. No Node.js. No CGO by default. A single Go binary serves everything.

---

## Contents

- [What this is](#what-this-is)
- [How it works](#how-it-works)
- [Installation](#installation)
- [Quick start](#quick-start)
- [CLI reference](#cli-reference)
  - [Global flags](#global-flags)
  - [`pola new`](#pola-new) — scaffold a new app
  - [`pola dev`](#pola-dev) — start the dev server
  - [`pola build`](#pola-build) — build a production binary
  - [`pola generate`](#pola-generate) — code generators
    - [`generate action`](#pola-generate-action)
    - [`generate js:bridge`](#pola-generate-jsbridge)
    - [`generate migration`](#pola-generate-migration)
    - [`generate model`](#pola-generate-model)
    - [`generate page`](#pola-generate-page)
    - [`generate repository`](#pola-generate-repository)
    - [`generate route`](#pola-generate-route)
    - [`generate scaffold`](#pola-generate-scaffold)
    - [`generate service`](#pola-generate-service)
    - [`generate storage`](#pola-generate-storage)
    - [`generate zod`](#pola-generate-zod)
  - [`pola db`](#pola-db) — database management
    - [`db migrate`](#pola-db-migrate)
    - [`db rollback`](#pola-db-rollback)
    - [`db status`](#pola-db-status)
    - [`db reset`](#pola-db-reset)
    - [`db schema:load`](#pola-db-schemaload)
- [Configuration: `Polafile.hcl`](#configuration-polafilehcl)
- [Environment variables](#environment-variables)
- [Project structure](#project-structure)
- [File conventions](#file-conventions-nextjs-app-router)
- [Server vs. Client components](#server-vs-client-components)
- [Go ↔ JS bridge (JSI)](#go--js-bridge-jsi)
- [Hot reload](#hot-reload)
- [Selecting a JS VM](#selecting-a-js-vm)
- [Field syntax for generators](#field-syntax-for-generators)
- [Architecture decisions](#architecture-decisions)
- [Dependencies](#dependencies)

---

## What this is

Pola lets you write React Server Components in TSX that run inside a Go process. Go functions are exposed to the JS runtime via a typed bridge (`JSI`), so your components can call your database, cache, or any Go service directly — no API layer required.

The server streams output using the **RSC Flight Protocol** — React's native wire format. Suspense boundaries resolve concurrently and stream their content as they complete.

```tsx
// app/posts/page.tsx — runs in Go's JS VM, not in Node.js
import JSI from "@pola/jsi"

export default async function PostsPage() {
  const posts = await JSI.getPosts()  // ← calls a Go function directly
  return (
    <ul>
      {posts.map(p => <li key={p.slug}>{p.title}</li>)}
    </ul>
  )
}
```

---

## How it works

### Request lifecycle

```
Browser
  │
  ▼  GET /posts
Go net/http
  │
  ├─ Router matches route → builds props {params, searchParams}
  ├─ Acquires VM from pool
  ├─ Injects per-request context + JSI bridge into VM
  │
  ▼  VM calls __render__(exportName, propsJSON)
JS VM (Goja / QuickJS / V8 / ...)
  │
  ├─ renderToReadableStream(Page, props, clientManifest)
  ├─ Server Components run synchronously / via async await
  ├─ Client components → ClientRef (never executed server-side)
  │
  ▼  RSC Flight Protocol (chunked HTTP)
 0:["$","ul",null,{"children":[...]}]
 1:I{"id":"button-abc","chunks":["chunk-1.js"]}
  │
  ▼
Browser _client.js
  ├─ createFromFetch() parses Flight stream
  ├─ Hydrates with React DOM
  └─ Client components lazy-loaded from manifest
```

---

## Installation

### Build the CLI from source

```bash
git clone https://github.com/polagonow/pola
cd pola
go build -o bin/pola ./cmd/pola
sudo mv bin/pola /usr/local/bin/   # optional: install globally
```

### Run directly with `go run`

```bash
go run ./cmd/pola <command> [flags]
```

### Requirements

- Go 1.24 or later
- Node.js **not** required for runtime (Pola serves with a single Go binary). It is required at scaffolding time only to install JS dependencies (`npm`/`pnpm`/`yarn`).
- CGO not required by default. Some VM backends (`v8go`, `quickjsgo`, `moderncquickjs`) need CGO; the default `goja` does not.

---

## Quick start

```bash
# 1. Scaffold a new app
pola new my-app

# 2. Move in and start the dev server
cd my-app
pola dev                    # http://localhost:3000

# 3. Generate a CRUD scaffold
pola generate scaffold Post title:string body:text

# 4. Run migrations
pola db migrate

# 5. Build for production
pola build -o ./bin/myapp
./bin/myapp
```

---

## CLI reference

The top-level help:

```
pola [command] [flags]

Available Commands:
  build       Build a production binary
  db          Database management commands
  dev         Start the development server (alias: serve)
  generate    Generate bridge code from actions/ directory (aliases: gen, g)
  help        Help about any command
  new         Create a new Pola application
```

### Global flags

These flags work on every command:

| Flag | Description |
|------|-------------|
| `-v`, `--verbose` | Enable verbose output |
| `--cwd <dir>` | Run as if pola was started in this directory |
| `-h`, `--help` | Help for the current command |
| `--version` | Print the version of `pola` |

---

### `pola new`

Scaffold a new Pola application with a working project structure, including a Go server entry point, React app directory, and configuration files.

**Usage**

```
pola new <app-name> [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--renderer` | `react` | View renderer (`react`) |
| `--bundler` | `esbuild` | JS bundler (`esbuild`) |
| `--router` | `nextjs` | Router style (`nextjs`) |
| `--css` | `tailwind` | CSS processor (`tailwind`, `sass`, `none`) |
| `--vm` | `goja` | JS engine (`goja`) |
| `--ui` | `none` | UI library — one of `shadcn`, `mui`, `slds`, `ads`, `carbon`, `patternfly`, `fluentui`, `antd`, `none` |
| `--pm` | auto | JS package manager (`npm`, `pnpm`, `yarn`); auto-detected if not set |
| `--csrf` | `true` | Enable CSRF protection |
| `--security-headers` | `true` | Enable security headers |
| `--pola-path` | — | Local path to pola framework source (adds a `replace` directive — for development against an unpublished pola) |

**UI compatibility rules**

- `--ui=shadcn` requires `--css=tailwind` and `--renderer=react`.
- `--ui=slds`, `--ui=ads`, `--ui=carbon`, `--ui=fluentui`, `--ui=antd` all require `--renderer=react`.
- The CSS processor is auto-adjusted based on the UI choice (e.g. SLDS pulls in Sass).

**Examples**

```bash
pola new my-app
pola new my-app --renderer=react --bundler=esbuild
pola new my-app --css=tailwind --ui=shadcn
pola new admin --ui=antd --pm=pnpm
pola new local-dev --pola-path=../pola              # develop against local pola checkout
```

**What it does**

1. Creates the target directory and scaffolds files from the chosen renderer/UI templates.
2. Writes a [`Polafile.hcl`](#configuration-polafilehcl) locking your choices and their module versions.
3. Runs `go mod tidy` to resolve plugin dependencies.
4. Runs `<pm> install` in `web/` to install JS dependencies.
5. Stubs the `@pola/actions` and `@pola/react` packages into `node_modules`.
6. Runs the `js:bridge` generator to produce TypeScript declarations.

---

### `pola dev`

Run the Pola app in development mode with hot reload. Watches `.go` and `.tmpl` files and re-runs the server on change; watches `.tsx`/`.ts` files through the framework's own watcher and pushes browser reload over WebSocket.

**Usage**

```
pola dev [flags]
```

**Aliases:** `serve`

**Flags**

| Flag | Default (env) | Description |
|------|---------------|-------------|
| `-p`, `--port` | `3000` (`PORT`) | Server port |
| `--renderer` | `react` (`POLA_RENDERER`) | View renderer |
| `--bundler` | `esbuild` (`POLA_BUNDLER`) | JS bundler |
| `--router` | `nextjs` (`POLA_ROUTER`) | Router style |
| `--css` | `tailwind` (`POLA_CSS`) | CSS processor |
| `--vm` | `goja` (`POLA_VM`) | JS engine |
| `--app-path` | `./web` (`POLA_WEBAPP_PATH`) | Path to the web app directory |
| `--csrf` | `true` (`POLA_CSRF`) | Enable CSRF protection |
| `--security-headers` | `true` (`POLA_SECURITY_HEADERS`) | Enable security headers |

Defaults fall back through: CLI flag → env var → `Polafile.hcl` → hardcoded default.

**Examples**

```bash
pola dev
pola dev --port 8080
pola dev --vm goja --css tailwind
PORT=4000 pola dev
pola dev --app-path ./frontend
```

---

### `pola build`

Build a production-ready single binary in two stages:

1. **Bundle stage** — runs your app with `POLA_BUILD_ONLY=true` to produce JS/CSS assets in `./public`.
2. **Compile stage** — compiles a Go binary with embedded assets and `-ldflags="-s -w"`.

**Usage**

```
pola build [flags]
```

**Flags**

| Flag | Default (env) | Description |
|------|---------------|-------------|
| `-o`, `--output` | `./bin/<app-name>` | Output binary path |
| `--renderer` | `react` (`POLA_RENDERER`) | View renderer |
| `--bundler` | `esbuild` (`POLA_BUNDLER`) | JS bundler |
| `--router` | `nextjs` (`POLA_ROUTER`) | Router style |
| `--css` | `tailwind` (`POLA_CSS`) | CSS processor |
| `--vm` | `goja` (`POLA_VM`) | JS engine |
| `--cgo` | `1` (`CGO_ENABLED`) | Value of `CGO_ENABLED` to pass to `go build` |
| `--app-path` | `./web` (`POLA_WEBAPP_PATH`) | Path to the web app directory |
| `--csrf` | `true` (`POLA_CSRF`) | Enable CSRF protection |
| `--security-headers` | `true` (`POLA_SECURITY_HEADERS`) | Enable security headers |

**Examples**

```bash
pola build
pola build -o ./bin/myapp
pola build --vm goja --renderer react
CGO_ENABLED=0 pola build --vm goja            # fully static binary
pola build --cgo 1 --vm v8go                  # build with V8
```

---

### `pola generate`

Generate code from `actions/` and scaffolds for common resources. Without a subcommand, runs the bridge generator and prints an overlay path for `go run -overlay`.

**Usage**

```
pola generate [flags]               # bridge codegen + overlay
pola generate <subcommand> [args]   # scaffold
```

**Aliases:** `gen`, `g`

**Common flags (inherited by every subcommand)**

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Overwrite files that already exist |
| `--skip-collision-check` | `false` | Skip the file-exists check entirely |
| `--actions-dir` | `./actions` | Path to actions directory |
| `--ts-out` | — | Path to the generated TypeScript `.d.ts` file |

#### `pola generate action`

Scaffold a new action struct with boilerplate methods.

**Usage**

```
pola generate action <Name> [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--service` | — | Wire action methods to the named service |

**Examples**

```bash
pola generate action Blog
pola generate action Products
pola generate action Products --service=Product
```

#### `pola generate js:bridge`

Parse Go action structs and write TypeScript declarations so client-side code gets typed `di` imports. No flags. Equivalent to the implicit codegen step run on `pola new` and `pola dev`.

**Example**

```bash
pola generate js:bridge
```

#### `pola generate migration`

Diff ORM model schemas against the migration directory and auto-generate a versioned SQL migration file using Atlas. The migration is written to the directory configured in `Polafile.hcl` (`database.migrations.directory`).

**Usage**

```
pola generate migration <name> [flags]
```

**Aliases:** `mi`

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `development` | Environment to use for adapter/dev-url resolution |
| `--dev-url` | — | Dev database URL (overrides Polafile and auto-detect) |

**Examples**

```bash
pola generate migration CreateUsers
pola generate migration AddEmailToUsers --env development
pola generate migration init --dev-url "sqlite://file?mode=memory"
```

#### `pola generate model`

Generate an ORM model/schema from field definitions. The ORM is read from `Polafile.hcl` (`database.orm`, e.g. `ent` or `gorm`).

**Usage**

```
pola generate model <Name> [field:type ...] [flags]
```

**Aliases:** `m`

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--skip-migration` | `false` | Skip auto-generating a migration after model creation |

**Examples**

```bash
pola generate model User name:string email:string:uniq age:int
pola generate model Article title:string:index body:text author:references
pola generate model Comment body:text commentable:references{polymorphic}
pola generate model User name:string avatar:references{StorageBlob}
```

See [Field syntax](#field-syntax-for-generators) for the full grammar.

#### `pola generate page`

Scaffold CRUD pages (list, show, create, edit) for a resource using the renderer from `Polafile.hcl`. Field syntax matches the model generator.

**Usage**

```
pola generate page <Name> [field:type ...]
```

**Aliases:** `p`

**Examples**

```bash
pola generate page Product name:string price:float description:text
pola generate page Article title:string body:text
```

#### `pola generate repository`

Generate a repository interface and an ORM-specific implementation. The ORM is read from `Polafile.hcl`.

**Usage**

```
pola generate repository <Name> [field:type ...]
```

**Aliases:** `repo`

**Examples**

```bash
pola generate repository User name:string email:string:uniq
pola generate repository Product name:string price:float
```

#### `pola generate route`

Create a new route file in the `routes/` directory with HTTP method stubs. Methods can be passed as separate args or comma-separated. Default method is `GET`.

**Usage**

```
pola generate route <Name> [methods...] [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--service` | — | Wire route handlers to the named service via DI |

**Examples**

```bash
pola generate route Posts
pola generate route Posts GET,POST
pola generate route Posts/Comments GET POST DELETE
pola generate route Posts GET,POST,DELETE --service=Post
```

#### `pola generate scaffold`

Generate a full resource: model, repository, service, action, route, Zod schema, and CRUD pages.

**Usage**

```
pola generate scaffold <Name> [field:type ...] [flags]
```

**Aliases:** `s`

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--skip-model` | `false` | Skip model generation |
| `--skip-repository` | `false` | Skip repository generation |
| `--skip-service` | `false` | Skip service generation |
| `--skip-action` | `false` | Skip action generation |
| `--skip-route` | `false` | Skip route generation |
| `--skip-zod` | `false` | Skip Zod schema generation |
| `--skip-views` | `false` | Skip page generation |
| `--skip-migration` | `false` | Skip migration generation |

**Examples**

```bash
pola generate scaffold Product name:string price:float description:text
pola generate scaffold Product name:string --skip-route
pola generate scaffold Product name:string --skip-repository --skip-service
```

#### `pola generate service`

Generate a service struct that depends on a repository interface.

**Usage**

```
pola generate service <Name> [field:type ...]
```

**Aliases:** `svc`

**Examples**

```bash
pola generate service User name:string email:string
pola generate service Product name:string price:float
```

#### `pola generate storage`

Generate `StorageBlob` and `StorageAttachment` models and configure the storage block in `Polafile.hcl`.

- `StorageBlob` — file metadata (key, filename, content type, size, checksum).
- `StorageAttachment` — polymorphic join table that links any model to blobs.

**Usage**

```
pola generate storage [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--driver` | `fs` | Storage driver: `fs` or `rclone` |
| `--root` | `uploads` | Storage root path (local dir for `fs`, `remote:path` for `rclone`) |
| `--config-path` | — | Path to rclone config file (rclone driver only) |

**Examples**

```bash
pola generate storage
pola generate storage --driver fs --root ./uploads
pola generate storage --driver rclone --root myremote:bucket/path
pola generate storage --driver rclone --root myremote:bucket/path --config-path /etc/rclone/rclone.conf
```

After running, associate files with models:

```bash
# Direct: User has one avatar blob
pola generate model User name:string avatar:references{StorageBlob}

# Polymorphic: any model can have attachments via StorageAttachment
```

#### `pola generate zod`

Generate a TypeScript Zod schema for a resource. Written to `app/schemas/`.

**Usage**

```
pola generate zod <Name> [field:type ...]
```

**Aliases:** `z`

**Examples**

```bash
pola generate zod Product name:string price:float description:text
pola generate zod Article title:string body:text
```

---

### `pola db`

Database management. Connection details come from `Polafile.hcl` (`pola.database`) merged with the `--env` block, or you can pass `--url` to override.

> Currently only **SQLite** is supported by the migration runner. PostgreSQL/MySQL adapters are wired through `Polafile.hcl` for app runtime but `db migrate/rollback/...` will error if the adapter is not sqlite.

#### `pola db migrate`

Apply all pending migrations to the database. With `--version`, migrate to a specific version.

**Usage**

```
pola db migrate [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `development` | Polafile environment block to merge |
| `--url` | — | Database URL (overrides Polafile) |
| `--version` | — | Migrate to a specific version |

**Examples**

```bash
pola db migrate
pola db migrate --url "sqlite:dev.db"
pola db migrate --version 20240101120000
pola db migrate --env production
```

#### `pola db rollback`

Rollback the last applied migration. Uses `-- atlas:down` directives in each migration file to know what SQL to execute.

**Usage**

```
pola db rollback [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `development` | Polafile environment block |
| `--url` | — | Database URL (overrides Polafile) |
| `--step` | `1` | Number of migrations to rollback |

**Examples**

```bash
pola db rollback
pola db rollback --step 3
pola db rollback --url "sqlite:dev.db"
```

#### `pola db status`

Show applied and pending migrations in a table.

**Usage**

```
pola db status [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `development` | Polafile environment block |
| `--url` | — | Database URL (overrides Polafile) |

**Examples**

```bash
pola db status
pola db status --url "sqlite:dev.db"
```

Output:

```
VERSION          DESCRIPTION       STATUS    APPLIED AT
-------          -----------       ------    ----------
20240101120000   create_users      applied   2026-04-01T10:15:00Z
20240115093000   add_email_index   pending   -

Applied: 1, Pending: 1
```

#### `pola db reset`

Drop **all tables** in the database, then re-run every migration from scratch. Destructive.

**Usage**

```
pola db reset [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `development` | Polafile environment block |
| `--url` | — | Database URL (overrides Polafile) |

**Examples**

```bash
pola db reset
pola db reset --url "sqlite:dev.db"
```

#### `pola db schema:load`

Apply all migration files to a fresh database. Useful for setting up a new environment.

**Usage**

```
pola db schema:load [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--env` | `development` | Polafile environment block |
| `--url` | — | Database URL (overrides Polafile) |

**Examples**

```bash
pola db schema:load
pola db schema:load --url "sqlite:dev.db"
```

---

## Configuration: `Polafile.hcl`

`Polafile.hcl` is generated at the project root when running `pola new`. It locks user choices and acts as the source of truth for every subsequent `pola` command. Uses [HashiCorp HCL](https://github.com/hashicorp/hcl) syntax.

```hcl
pola {
  package         = "my-app"
  renderer        = "react"
  engine          = "goja"
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"

  csrf             { enabled = true }
  security_headers { enabled = true }

  cache {
    enabled = true
    adapter = "memory"

    env "production" {
      adapter  = "redis"
      host     = "localhost"
      port     = "6379"
    }
  }

  database {
    models = "db/models"
    orm    = "gorm"

    migrations {
      directory = "db/migrations"
      format    = "sql"
    }

    env "development" { adapter = "sqlite" }
    env "production"  { adapter = "postgresql" }
  }
}
```

### Top-level attributes

All attributes are optional.

| Attribute | Description | Example |
|-----------|-------------|---------|
| `package` | Go module name (set by `pola new`) | `my-app` |
| `version` | Pola CLI version used to create the project | `0.1.0` |
| `renderer` | View renderer | `react` |
| `engine` | JavaScript engine | `goja` |
| `bundler` | JS bundler | `esbuild` |
| `router` | Router style | `nextjs` |
| `css` | CSS processor | `tailwind`, `sass`, `none` |
| `ui` | UI library | `shadcn`, `antd`, `none` |
| `package_manager` | JS package manager | `pnpm`, `npm`, `yarn` |
| `app` | Directory for frontend app | `web` |
| `actions` | Directory for server actions | `actions` |
| `routes` | Directory for API routes | `routes` |
| `repositories` | Directory for repositories | `repositories` |
| `services` | Directory for services | `services` |

### Nested blocks

| Block | Purpose | Per-env override |
|-------|---------|------------------|
| `csrf { enabled }` | CSRF protection | yes |
| `security_headers { enabled }` | Security headers | yes |
| `cache { adapter, host, port, password, db }` | Cache backend | yes |
| `database { url, dev_url, models, adapter, orm }` | Database connection + ORM | yes |
| `database > migrations { directory, format }` | Migration files | no |

### Resolution order

For any setting (e.g. `bundler`, `vm`, `port`), the CLI resolves in this order — **first match wins**:

1. Explicit CLI flag (`--bundler=webpack`)
2. Environment variable (`POLA_BUNDLER`)
3. `Polafile.hcl`
4. Hardcoded default

---

## Environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `PORT` | Dev server port | `3000` |
| `POLA_RENDERER` | View renderer | `react` |
| `POLA_BUNDLER` | JS bundler | `esbuild` |
| `POLA_ROUTER` | Router style | `nextjs` |
| `POLA_CSS` | CSS processor | `tailwind` |
| `POLA_VM` | JS engine | `goja` |
| `POLA_CACHE` | Cache adapter | `memory` |
| `POLA_CSRF` | Enable CSRF (`false` disables) | `true` |
| `POLA_SECURITY_HEADERS` | Enable security headers | `true` |
| `POLA_WEBAPP_PATH` | Path to the web app dir | `./web` |
| `POLA_PM` | JS package manager | autodetect |
| `POLA_ENV` | Environment label exposed to runtime | `development` (in `pola dev`) |
| `POLA_BUILD_ONLY` | Set by `pola build` stage 1; tells runtime to bundle & exit | — |
| `POLA_PUBLIC_DIR` | Output directory for bundle stage 1 | `./public` |
| `CGO_ENABLED` | Forwarded to `go build` | `1` |

---

## Project structure

A freshly scaffolded project (`pola new my-app`):

```
my-app/
├── Polafile.hcl              Project configuration (engine, ORM, etc.)
├── go.mod
├── main.go                   `pola.Ready()` then `http.ListenAndServe`
├── actions/                  Go structs exposed to the JS VM (typed bridge)
├── routes/                   HTTP route handlers
├── repositories/             Data-access layer (DB interfaces + impls)
├── services/                 Business-logic layer
├── db/
│   ├── models/               GORM/Ent models
│   └── migrations/           Versioned SQL migrations (Atlas format)
├── public/                   Static assets (favicon, embedded bundles)
└── web/                      Frontend (Next.js-style app/ directory)
    ├── app/                  Pages, layouts, error boundaries
    ├── components/
    ├── package.json
    └── tsconfig.json
```

---

## File conventions (Next.js App Router)

Pages live under the `app/` directory inside `web/`. Pola discovers them automatically:

| File | Route |
|------|-------|
| `app/page.tsx` | `/` |
| `app/posts/page.tsx` | `/posts` |
| `app/posts/[slug]/page.tsx` | `/posts/:slug` |
| `app/posts/[...path]/page.tsx` | `/posts/:...path` (catch-all) |
| `app/posts/[[...path]]/page.tsx` | `/posts/:...path?` (optional catch-all) |

**Companion files** (all optional):

| File | Purpose |
|------|---------|
| `layout.tsx` | Wraps children for this segment and below |
| `error.tsx` | Error boundary for this segment |
| `loading.tsx` | Suspense fallback while page loads |
| `not-found.tsx` | Per-segment 404 |
| `global-error.tsx` | Top-level error boundary |
| `global-not-found.tsx` | Fallback 404 |

---

## Server vs. Client components

### Server Components

Any `.tsx` file without `"use client"` is a Server Component. It runs in the VM on every request.

```tsx
// app/posts/[slug]/page.tsx
import JSI from "@pola/jsi"

interface Props { params: { slug: string } }

export default async function PostPage({ params }: Props) {
  const post = await JSI.getPost(params.slug)
  return (
    <article>
      <h1>{post.title}</h1>
      <div dangerouslySetInnerHTML={{ __html: post.body }} />
    </article>
  )
}
```

### Client Components

Files with `"use client"` at the top are bundled for the browser. The server never executes them — it emits a `ClientRef` into the Flight stream, and the browser resolves it from `manifest.json`.

```tsx
"use client"
// app/components/LikeButton.tsx
import { useState } from "react"

export default function LikeButton({ postId }: { postId: string }) {
  const [liked, setLiked] = useState(false)
  return (
    <button onClick={() => setLiked(true)}>
      {liked ? "Liked" : "Like"}
    </button>
  )
}
```

---

## Go ↔ JS bridge (JSI)

Pola generates a typed bridge from Go structs in `actions/`. Methods on those structs become `JSI.*` calls available inside Server Components.

```go
// actions/posts.go
type Posts struct{ DB *gorm.DB }

func (p *Posts) GetPosts() ([]Post, error) {
    var posts []Post
    return posts, p.DB.Find(&posts).Error
}

func (p *Posts) GetPost(slug string) (*Post, error) {
    var post Post
    return &post, p.DB.First(&post, "slug = ?", slug).Error
}
```

Then in TSX:

```tsx
import JSI from "@pola/jsi"

const posts = await JSI.getPosts()
const post  = await JSI.getPost("hello-world")
```

The bridge is regenerated automatically on `pola dev`, `pola build`, and `pola new`. Run `pola generate js:bridge` to refresh manually.

---

## Hot reload

In `pola dev`, two watchers run simultaneously:

- **Go watcher** — polls `.go`, `.tmpl`, `go.mod`, `go.sum`, `Polafile.hcl`. On change, kills and respawns the Go process.
- **JS watcher** — `fsnotify` on the `AppDir`. On `.tsx`/`.ts` change, re-runs discovery + bundling and pushes a reload event over `/__dev__/hot` (WebSocket).

The WebSocket client script is injected automatically into the HTML shell.

---

## Selecting a JS VM

The active VM is set by the `engine` field in `Polafile.hcl` (or `--vm` / `POLA_VM`). Available engines:

| VM | CGO | Notes |
|----|-----|-------|
| `goja` | No | **Default.** Pure-Go ES2020, EventLoop |
| `sobek` | No | Goja fork with improvements |
| `qjs` | No | QuickJS compiled to WASM |
| `quickjsgo` | Yes | QuickJS native binding |
| `moderncquickjs` | Yes | Modern QuickJS binding |
| `v8go` | Yes | V8 engine; fastest for CPU-heavy JS |

```bash
pola dev --vm v8go
POLA_VM=qjs pola build
```

---

## Field syntax for generators

The generators that accept fields (`model`, `repository`, `service`, `scaffold`, `zod`, `page`) all use the same syntax:

```
field:type{options}:modifier1:modifier2 ...
```

**Types**: `string`, `int`, `int64`, `float`, `bool`, `time`, `uuid`, `text`, `bytes`, `json`, `references`

**Options** (only on `references`):
- `{ModelName}` — points to a specific model (e.g. `avatar:references{StorageBlob}`)
- `{polymorphic}` — polymorphic association (e.g. `commentable:references{polymorphic}`)

**Modifiers**: `index`, `uniq`

**Examples**

```bash
pola generate model User \
  name:string \
  email:string:uniq \
  age:int

pola generate model Article \
  title:string:index \
  body:text \
  author:references

pola generate model Comment \
  body:text \
  commentable:references{polymorphic}

pola generate scaffold Product \
  name:string:index \
  sku:string:uniq \
  price:float \
  description:text
```

---

## Architecture decisions

**Why Goja over V8go by default?** Goja is pure Go — no CGO, no C++ toolchain, single static binary. V8go is faster for CPU-heavy JS but the CGO overhead and deployment complexity don't pay off for I/O-bound server rendering. The pluggable VM architecture means you can switch if you need to.

**Why implement the Flight Protocol ourselves?** The official `react-server-dom-*` packages are Node.js-only. The protocol is simple enough to implement in ~150 lines of Go, giving full control over streaming behaviour with no Node.js subprocess or IPC overhead.

**Why esbuild as a Go package?** esbuild exposes its full API as an importable Go package. The bundler is part of the binary — no separate build step, no `package.json` at the project root.

**Why Next.js file conventions?** They're well-understood and tooled. The discovery layer is an interface — you can replace it with your own convention.

**Why pluggable interfaces everywhere?** Swapping the VM, bundler, router, or renderer changes one Polafile field. The framework core has zero knowledge of any implementation.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/dop251/goja` | Pure-Go JS/ES2020 runtime |
| `github.com/dop251/goja_nodejs` | EventLoop for async/await in Goja |
| `github.com/evanw/esbuild` | Go-native JS/TSX bundler |
| `github.com/fsnotify/fsnotify` | Cross-platform file watcher (hot reload) |
| `github.com/gorilla/websocket` | WebSocket server (hot reload client push) |
| `github.com/spf13/cobra` | CLI command framework |
| `github.com/hashicorp/hcl/v2` | Polafile parser |
| `ariga.io/atlas` | Migration engine |

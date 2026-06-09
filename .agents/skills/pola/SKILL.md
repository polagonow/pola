---
name: pola
description: Build web applications WITH the Pola Go framework and its CLI — scaffold apps, generate models/repositories/services/actions/routes/pages, wire the Go↔JS (React Server Components) bridge, configure Polafile.hcl, run migrations, expose services as MCP tools, and ship a single static binary. Use when creating or extending an app built on pola (pola new / pola generate / pola dev / pola build), NOT when modifying the pola framework itself (for that, see the add-vm / add-bundler / add-renderer skills).
---

# Building apps with Pola

Pola is a Go framework for **React Server Components**. Your `.tsx` pages render
**inside a Go JS VM** (goja by default), stream to the browser over React's **Flight
protocol**, and call your Go code directly through a typed bridge. The whole app — server,
renderer, bundler, JS runtime — compiles to **one static Go binary**. No Node.js at runtime.

## When to use this skill

Use it when you are **building or extending an application with pola**:

- "Scaffold a new pola app" / `pola new`
- "Generate a model / scaffold / CRUD resource" / `pola generate …`
- "Expose this Go function to my React page" (the actions bridge)
- "Add an MCP tool / file uploads / a migration"
- "Configure `Polafile.hcl`" / "run `pola dev` or `pola build`"

**Do not** use it to *extend the framework itself* (adding a new VM, bundler, renderer,
polyfill). Those are separate contributor skills: `add-vm`, `add-bundler`, `add-renderer`,
`add-polyfill`, `add-e2e-test`.

## Mental model

```
Browser
  │  GET /posts
  ▼
Go net/http  ──► Router matches app/posts/page.tsx, builds {params, searchParams}
  │
  ▼  render inside a pooled JS VM (goja / sobek / v8 / quickjs)
React Server Component (page.tsx)
  │  const posts = await Blog.getPosts()   ◄── typed bridge into Go
  ▼
actions/ (Go struct)  ──►  services/  ──►  repositories/  ──►  DB (gorm/ent)
  │
  ▼  RSC Flight stream (chunked HTTP)  ──►  browser hydrates, lazy-loads "use client" parts
```

Key facts that shape every decision:

- **`main.go` is fixed boilerplate.** A pola app's `main.go` is *only* this — never add routing
  or handler logic here; the framework wires everything via `pola.Ready()`:
  ```go
  package main

  import (
      "log"
      "net/http"

      "myapp" // your module path
  )

  func main() {
      if err := pola.Ready(); err != nil {
          log.Fatal(err)
      }
      log.Fatal(http.ListenAndServe(pola.Addr(), nil))
  }
  ```
- **Server vs client components.** A `.tsx` file is a *Server Component* (runs in the VM, can
  call the bridge + `await`) unless its first line is `"use client"`, which makes it a browser
  component (no bridge calls; gets hydrated).
- **Where logic lives:** data exposed to React → `actions/`; HTTP/JSON endpoints → `routes/`;
  business logic → `services/`; data access → `repositories/`; ORM models → `db/models/`.

## The `@pola/actions` bridge — the one convention you must get right

Exported Go structs in `actions/` are exposed to React. **This is the #1 source of broken
generated code, so follow the verified convention exactly.**

Go side (`actions/blog.go`):
```go
package actions

type Blog struct{}

type Post struct {
    Slug  string `json:"slug"`
    Title string `json:"title"`
}

// Each exported method becomes a JS call. Must return (T, error) or just error.
func (b *Blog) GetPosts() ([]Post, error) { /* ... */ }
func (b *Blog) GetPost(slug string) (*Post, error) { /* ... */ }
```

TSX side (a Server Component):
```tsx
import { Blog } from "@pola/actions";        // named import = the struct name

export default async function PostsPage() {
  const posts = await Blog.getPosts();        // method names are camelCased
  return <ul>{posts.map((p) => <li key={p.slug}>{p.title}</li>)}</ul>;
}
```

Navigation uses the bridge's Link:
```tsx
import { Link } from "@pola/react/link";
```

`@pola/react` also ships these client helpers (stubbed into `node_modules` by the CLI):

```tsx
import { Image } from "@pola/react/image";     // optimized <Image>, routes through /_pola/image
import { useRouter, usePathname } from "@pola/react/router"; // Next.js-style client hooks
import { ImageResponse } from "@pola/react/og"; // dynamic OG image serializer
```

Rules:

- **Import the struct by name from `@pola/actions`** — e.g. `import { Blog } from "@pola/actions"`.
  The only bridge packages are `@pola/actions` (your action structs) and `@pola/react`
  (e.g. `Link`) — both stubbed into `node_modules` by the CLI. There is no default/global bridge
  object; always import the specific struct by name.
- **Methods are camelCased**: the leading run of capitals is lowercased — `GetPosts → getPosts`,
  `Get → get`, `URL → url`, `HTTPServer → httpServer`. Calls are **async** (return Promises).
- **Method signatures** must return `(T, error)` or `error`; params are positional and typed.
  In the generated TS types, Go `time.Time` and `error` become `string`, `*T` becomes `T | null`,
  `[]T` becomes `T[]`, `map[K]V` becomes `Record<K,V>`.
- **The bridge is generated, not hand-written.** It is injected via a Go `-overlay` at run/build
  time and lives in a temp dir — there is no bridge `.go` file in your project to edit.
  `@pola/actions` and `@pola/react` are stub packages in `node_modules`. After you add or change
  an action, run **`pola generate`** (bare) to refresh the TypeScript declarations.
- **Dependency injection:** an action may take constructor deps, e.g.
  `func NewProductAction(svc *services.ProductService) *ProductAction`. The framework resolves
  and wires them automatically — you don't instantiate actions yourself.

## Canonical end-to-end workflow

```bash
# 1. Scaffold (prompts for a name if omitted; runs go mod tidy + installs JS deps + stubs @pola/*)
pola new myapp --css tailwind --ui shadcn
cd myapp

# 2. Generate a full CRUD resource: model + repository + service + action + route + zod + pages
pola generate scaffold Post title:string:index body:text published:bool

# 3. Apply migrations (the migration runner is sqlite-only — see Pitfalls)
pola db migrate

# 4. Develop with hot reload (alias: `pola serve`)
pola dev                      # http://localhost:3000

# 5. Ship a single static binary (two-stage: bundle assets, then compile)
pola build -o bin/myapp
./bin/myapp
```

`pola new` is the only step that needs Node.js (to install JS deps). Everything after runs from
the Go binary.

## Project layout

A scaffolded app (`pola new myapp`):

```
myapp/
├── Polafile.hcl          source of truth: renderer, engine, ORM, blocks (see below)
├── go.mod
├── main.go               pola.Ready(); http.ListenAndServe(pola.Addr(), nil)  — don't touch
├── actions/              Go structs exposed to React via @pola/actions (the bridge)
├── routes/               HTTP/JSON endpoints — routes/<path>/route.go with GET/POST/… methods
├── services/             business logic; depend on repository interfaces
├── repositories/         data access: interface + ORM impl (repositories/gorm/…) + pagination.go
├── db/
│   ├── models/           ORM models (db/models/gorm/… or db/models/schema/… for ent)
│   └── migrations/       versioned migrations (Atlas)
├── mcp/                  optional MCP server: tools/ resources/ prompts/ (pola generate mcp …)
├── public/               static assets / embedded production bundle
└── web/                  the frontend (Next.js App Router style)
    ├── app/              pages: page.tsx, layout.tsx, [slug]/, (group)/, loading/error/not-found
    ├── components/       client + server components
    ├── schemas/          generated Zod schemas (pola generate zod)
    ├── package.json
    └── tsconfig.json
```

Data-layer chain: **`db/models` → `repositories/` (interface + ORM impl, paginated) → `services/`
(business logic) → `actions/` (React bridge) *or* `routes/` (HTTP).** See
`references/data-layer.md`.

## Polafile.hcl (the source of truth)

Generated by `pola new`; every `pola` command reads it. **All config nests inside a top-level
`pola { }` block** (keys are not bare at file root). Real example:

```hcl
pola {
  package         = "myapp"
  renderer        = "react"
  engine          = "goja"     # NOTE: the Polafile key is `engine`; the CLI flag is `--vm`
  bundler         = "esbuild"
  router          = "nextjs"
  css             = "tailwind"
  package_manager = "pnpm"
  app             = "web"
  actions         = "actions"
  routes          = "routes"

  csrf             { enabled = true }
  security_headers { enabled = true }
  cache            { enabled = true, adapter = "memory" }

  database {
    models = "db/models"
    orm    = "gorm"            # gorm | ent — generators read this
    migrations { directory = "db/migrations", format = "hcl" }
    env "development" { adapter = "sqlite" }
    env "production"  { adapter = "postgresql" }
  }
}
```

- **Resolution order (first match wins):** CLI flag → `POLA_*` env var → `Polafile.hcl` → default.
- **Per-environment overrides:** any block may contain `env "production" { … }` etc.
- **Available blocks:** `csrf`, `security_headers`, `cache`, `database` (+ `migrations`),
  `storage`, `mailer`, `image_processing`, `mcp`, `testing`.

Full schema, defaults, and env var names → `references/polafile.md`.

## Command & generator cheat-sheet

**Commands** (global flags: `-v/--verbose`, `--cwd <dir>`):

| Command | What it does |
|---------|--------------|
| `pola new [name]` | Scaffold a new app (flags: `--renderer --bundler --router --css --vm --ui --pm --module --csrf --security-headers --pola-path`) |
| `pola dev` (alias `serve`) | Dev server + hot reload (`-p/--port`, defaults from Polafile/env) |
| `pola build` | Two-stage production build → single binary (`-o`, `--cgo`) |
| `pola generate` (bare) | Regenerate the actions bridge + TS declarations |
| `pola generate <sub>` | Scaffold code (table below); aliases `gen`, `g` |
| `pola db <sub>` | `migrate` · `rollback` · `status` · `reset` · `schema:load` (**sqlite only**) |

**Generators** (`pola generate <x> [Name] [field:type …]`):

| Generator | Alias | Produces |
|-----------|-------|----------|
| `model` | `m` | ORM model in `db/models/…` (+ auto migration) |
| `repository` | `repo` | interface + ORM impl + `pagination.go` |
| `service` | `svc` | service struct depending on a repository |
| `action` | — | `actions/…` struct for the bridge (`--service` to wire one) |
| `route` | — | `routes/<path>/route.go` HTTP handlers (`--service`, file-upload aware) |
| `page` | `p` | renderer-specific CRUD pages in `web/app/…` + components |
| `scaffold` | `s` | **model + repository + service + action + route + zod + pages** |
| `zod` | `z` | Zod schema in `web/schemas/…` |
| `migration` | `mi` | Atlas migration by diffing models |
| `storage` | — | `StorageBlob` + `StorageAttachment` models + storage block |
| `mailer` | — | mailer struct + email templates (`<Name> [actions…]`) |
| `mcp` | — | `init` · `tool` · `resource` · `prompt` (MCP server artifacts) |
| `js:bridge` | — | TS declarations for `@pola/actions` (same as bare `pola generate`) |

**Field syntax** (shared by `model`/`repository`/`service`/`scaffold`/`zod`/`page`):

```
field:type{options}:modifier
```
- **types:** `string int int64 float bool time uuid text bytes json references`
- **trailing `?`** → optional/nullable (e.g. `bio:text?`)
- **`{…}` options** (mostly on `references`): `{ModelName}` target, `{polymorphic}`,
  `{N}` size for sized types (e.g. `name:string{255}`)
- **modifiers:** `index`, `uniq`

Full flag matrices → `references/cli.md`. Field grammar in depth → `references/data-layer.md`.

## Pitfalls (read before generating code)

1. **Bridge import is `@pola/actions`** — named struct exports (e.g.
   `import { Blog } from "@pola/actions"`), not a default or global object.
2. **Methods are camelCased and async.** `GetPosts → await Blog.getPosts()`. Actions return
   `(T, error)` or `error`. After editing actions, run `pola generate`.
3. **`pola db …` is sqlite-only.** It errors on postgres/mysql: *"adapter … not yet supported …
   currently only sqlite is supported"*. A common setup is `env "development" { adapter = "sqlite" }`
   / `env "production" { adapter = "postgresql" }`, but local `pola db migrate` runs against sqlite.
4. **The bridge is overlay-injected, not a file.** Don't look for or edit a generated bridge `.go`.
   `@pola/actions` / `@pola/react` are stubs in `node_modules`, refreshed by `pola new/dev/build/generate`.
5. **UI ↔ renderer/css rules.** Every `--ui` except `none` requires `--renderer=react`; `--ui=shadcn`
   also requires `--css=tailwind`. Some UIs auto-switch css to `sass` or `none`. Don't pair an
   incompatible UI with a non-react renderer.
6. **VM key duality.** The CLI flag is `--vm`; the Polafile key is `engine` (`engine = "goja"`).
   Don't write `vm = …` in the Polafile.
7. **CGO per VM.** `pola build` defaults `CGO_ENABLED=1`. `goja`/`sobek` are pure-Go (CGO-free);
   `v8go`/`quickjsgo`/`moderncquickjs` need CGO + a C toolchain. For a fully static binary, stay on
   goja/sobek and `CGO_ENABLED=0`.
8. **`main.go` stays boilerplate.** Routes go in `routes/`; data exposure goes in `actions/`.
9. **HCL config is wrapped in `pola { }`** — keys are not bare at the file root.

## References

Open these on demand:

- **`references/cli.md`** — every command, subcommand, and flag; exact generator outputs.
- **`references/polafile.md`** — full `Polafile.hcl` schema, all blocks, env overrides, `POLA_*` vars.
- **`references/conventions.md`** — directory layout, Next.js routing, server/client rules, naming,
  the camelCase rule in detail, DI.
- **`references/data-layer.md`** — field-spec grammar, the model→repo→service→action/route chain
  with real skeletons, ORM selection, migrations, storage/file uploads.
- **`references/recipes.md`** — full worked walkthroughs: blog CRUD via scaffold, an MCP tool over a
  service, and file uploads with storage.

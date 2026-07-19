# fumadocs-docs

A full [Fumadocs](https://www.fumadocs.dev/) documentation site running on
**Pola** — using the **real** `fumadocs-core` and `fumadocs-ui` npm packages, but
with **zero Node.js** at build or runtime. It compiles to a single Go binary and
serves as Pola's own documentation.

The content (`web/content/docs/**`) documents Pola itself: quick start,
architecture, routing, server actions, the data layer, migrations, auth, the CLI,
configuration, the pluggable stack, and deployment — with Markdown, GFM, syntax
highlighting, a table of contents, `> [!NOTE]` callouts, and `<Cards>`.

## How it works

Fumadocs' content pipeline (`fumadocs-mdx`) is a Node/Vite build tool, so it is
**replaced** by a Go-native MDX pipeline (`renderer/mdx`, built on `goldmark`).
Everything else is genuine Fumadocs:

- `.mdx` under `web/content/docs` is compiled to a React module by Go, via an
  esbuild `OnLoad` plugin (`renderer/react/esbuild/mdx.go`). The module exports
  `frontmatter` / `toc` / `structuredData` / a body component — the shape
  `fumadocs-core/source` expects.
- `fumadocs-core/source` `loader()` builds the page tree and runs **inside
  Pola's Go JS engine** (Goja) — it needs no Node built-ins.
- `fumadocs-ui` components are `"use client"`, so Pola emits them as RSC client
  references that hydrate in the browser; only the server parts run in Goja.
- Block components (`Callout`, `Cards`) are re-exported through a small
  `"use client"` shim (`@pola/fumadocs/blocks`) so their browser-only transitive
  deps — e.g. `lucide-react`, whose icon-name regex the pure-Go engine's RE2
  cannot compile — stay out of the server bundle. Result: the whole site renders
  on the **default goja engine, no CGO**.

## Commands used to build this example

```sh
# 1. Scaffold a React + Tailwind app (run from examples/, using the local framework)
go run ../cmd/pola new fumadocs-docs \
  --css tailwind --yes --skip-tests --csrf=false --security-headers=false

# 2. Point the module at the local pola checkout (monorepo example convention)
cd fumadocs-docs
go mod edit -replace github.com/polagonow/pola=../..

# 3. Add the real Fumadocs packages to web/package.json, then install
#    (fumadocs-core, fumadocs-ui, lucide-react — pinned to 16.11.5)
cd web && pnpm install

# 4. Run it
pola dev            # dev server at http://localhost:3000  (try /docs)
pola build          # single production binary, no Node.js
```

You can also scaffold this whole section into any Pola app with one command:

```sh
pola generate docs   # content/docs + /docs routes + fumadocs deps + Tailwind preset
```

## What works (verified)

- Multi-page docs at `/docs/[[...slug]]` with the full `fumadocs-ui` `DocsLayout`
  (sidebar, nav, breadcrumbs), driven by `fumadocs-core/source`'s `loader()`
  running server-side in Goja.
- Go-native MDX: frontmatter, GFM, table of contents, and **syntax highlighting**
  (chroma), compiled by `renderer/mdx` and loaded via an esbuild `.mdx` plugin.
- The `@pola/fumadocs` framework adapter maps fumadocs' routing/link onto Pola's
  client runtime; `fumadocs-ui`'s Tailwind preset styles the site.
- Unicode content (em dashes, curly quotes, accents) renders correctly.
- Single Go binary, **zero Node.js** at build or runtime; default Goja engine,
  no polyfills.

## Scope / limitations

- The Go MDX compiler targets Markdown + GFM + frontmatter + TOC + code
  highlighting. Arbitrary JSX/ESM inside `.mdx` is intentionally **not**
  supported — this is a Markdown compiler, not a general MDX transpiler. A
  bounded set of MDX components (`<Callout>`, `<Cards>`, …) is future work.
- Search is not yet wired (fumadocs-core static Orama search is future work).
- The body is injected via `dangerouslySetInnerHTML`, so per-element fumadocs-ui
  component overrides don't apply yet (a JSX-tree renderer is future work).

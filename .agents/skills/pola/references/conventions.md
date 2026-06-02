# Conventions: structure, routing, components, naming, DI

## Directory layout

```
myapp/
├── Polafile.hcl          configuration (see references/polafile.md)
├── go.mod
├── main.go               pola.Ready(); http.ListenAndServe(pola.Addr(), nil)  — fixed boilerplate
├── actions/              Go structs exposed to React (the @pola/actions bridge)
├── routes/               HTTP/JSON endpoints: routes/<path>/route.go
├── services/             business logic; depend on repository interfaces
├── repositories/         data access: <name>_repository.go (interface) + gorm/ (impl) + pagination.go
├── db/
│   ├── models/           ORM models: gorm/<name>.go  (ent: schema/<name>.go)
│   └── migrations/        Atlas migrations
├── mcp/                  optional: tools/ resources/ prompts/
├── public/               static assets / embedded production bundle
└── web/
    ├── app/              pages & layouts (Next.js App Router)
    ├── components/       client + server components
    ├── schemas/          generated Zod schemas
    ├── utils/            e.g. csrf.ts (generated for forms)
    ├── package.json
    └── tsconfig.json
```

Import alias: `tsconfig.json` maps `@/*` to the web app root, so `import X from "@/components/X"`.

## Routing (Next.js App Router, under `web/app/`)

Pola discovers pages automatically by file convention:

| File | Route |
|------|-------|
| `app/page.tsx` | `/` |
| `app/posts/page.tsx` | `/posts` |
| `app/posts/[slug]/page.tsx` | `/posts/:slug` (dynamic segment) |
| `app/posts/[...path]/page.tsx` | catch-all |
| `app/posts/[[...path]]/page.tsx` | optional catch-all |
| `app/(blog)/posts/page.tsx` | `/posts` — `(blog)` is a **route group**, omitted from the URL |

Companion files (all optional, per segment):

| File | Purpose |
|------|---------|
| `layout.tsx` | Wraps this segment and everything below it |
| `loading.tsx` | Suspense fallback while the page resolves |
| `error.tsx` | Error boundary for the segment |
| `not-found.tsx` | Per-segment 404 |
| `global-error.tsx` / `global-not-found.tsx` | Top-level fallbacks |

Page props: a page receives `{ params, searchParams }`. `params` carries dynamic segments,
`searchParams` is `Record<string, string>`:

```tsx
import { Blog } from "@pola/actions";

export default async function PostPage({
  params,
  searchParams,
}: {
  params: { slug: string };
  searchParams?: Record<string, string>;
}) {
  const post = await Blog.getPost(params.slug);
  return <h1>{post.title}</h1>;
}
```

## Server vs. client components

- **Server Component** (default): any `.tsx` without `"use client"`. Runs inside the Go VM on each
  request. Can be `async`, can call the `@pola/actions` bridge and `await` it. Has no browser state
  (no `useState`, no event handlers).
- **Client Component**: first line is `"use client"`. Bundled for the browser; the server emits a
  reference and the browser hydrates it. Use for interactivity (`useState`, `onClick`, etc.). It
  **cannot** call the bridge — fetch data in a server component and pass it down as props.

```tsx
"use client";
import { useState } from "react";

export default function LikeButton({ initialCount }: { initialCount: number }) {
  const [count, setCount] = useState(initialCount);
  return <button onClick={() => setCount(count + 1)}>♥ {count}</button>;
}
```

Navigation between routes uses the bridge's client Link:

```tsx
"use client";
import { Link } from "@pola/react/link";
// <Link href="/posts" className={({ isActive }) => isActive ? "..." : "..."}>Posts</Link>
```

## Naming and the bridge

- A pola app's Go module path is whatever `pola new` set (e.g. `myapp`); intra-app imports use it:
  `import "myapp/services"`.
- Generated files use snake_case: `product_action.go`, `product_service.go`,
  `product_repository.go`, `db/models/gorm/product.go`.
- **Each exported struct in `actions/` becomes a named export** from `@pola/actions`. The struct
  `ProductAction` → `import { ProductAction } from "@pola/actions"`.
- **Methods are camelCased for JS**, lowercasing the *leading run of uppercase letters*:

  | Go method | JS call |
  |-----------|---------|
  | `Get` | `get` |
  | `GetPosts` | `getPosts` |
  | `List` | `list` |
  | `URL` | `url` |
  | `HTTPServer` | `httpServer` |

  (A long leading run keeps its last capital as the start of the next word.)
- **Action methods must return `(T, error)` or `error`.** In the generated TS, the trailing `error`
  is dropped and the call returns `Promise<T>`. Type mapping: Go `time.Time` and `error` → `string`,
  `*T` → `T | null`, `[]T` → `T[]`, `map[K]V` → `Record<K, V>`, numeric Go types → `number`.
- After editing any `actions/` file, run **`pola generate`** to refresh the `.d.ts`.

## Dependency injection

The framework owns a DI container; you never instantiate actions/services/repositories yourself.

- **Actions/services/routes take constructor deps.** A service-backed action looks like:
  ```go
  type ProductAction struct{ svc *services.ProductService }
  func NewProductAction(svc *services.ProductService) *ProductAction { return &ProductAction{svc: svc} }
  ```
  Generators wire these when you pass `--service`. The autoload discovers `New<Name>…` constructors
  and registers them.
- **MCP tools resolve from the registry.** A DI-flavored tool takes `*core.Registry` and pulls
  collaborators out with `core.MustInvoke`:
  ```go
  func NewGreetingTool(r *core.Registry) *GreetingTool {
      return &GreetingTool{svc: core.MustInvoke[*services.GreetingService](r)}
  }
  ```

This is why the data chain composes cleanly: a repository impl takes `*gorm.DB`, a service takes the
repository interface, an action takes the service, and the bridge/autoload connects them at startup.

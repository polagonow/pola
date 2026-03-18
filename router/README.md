# router

File-system router implementations for the Pola framework.

Each router implements `core.Router`. It scans the app directory for page files
(using file extensions declared by the active renderer) and resolves incoming HTTP
requests to matched routes. The router is decoupled from the renderer.

## Available routers

| Package | Build tag | Status | Notes |
|---------|-----------|--------|-------|
| `router/nextjs` | `nextjs` | **FULL** | Next.js-style file-system routing (`[slug]`, `(group)`, `@parallel`) |
| `router/std` | `std` | stub | Standard Go path routing |
| `router/htmx` | `htmx` | stub | HTMX-specific routing |

## Route patterns (nextjs router)

| Pattern | Example | Description |
|---------|---------|-------------|
| Static | `app/about/page.tsx` | `/about` |
| Dynamic | `app/posts/[slug]/page.tsx` | `/posts/:slug` |
| Route group | `app/(work)/projects/page.tsx` | `/projects` (group ignored in URL) |
| Catch-all | `app/[...path]/page.tsx` | `/anything/deeply/nested` |
| Parallel | `app/@modal/page.tsx` | Parallel route slot |

## Build tags

```bash
go build -tags nextjs ./...       # nextjs (default)
POLA_ROUTER=nextjs mage build     # via Magefile env var
```

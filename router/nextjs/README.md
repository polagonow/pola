# Frontend Routes (Next.js Router)

File-system based router following Next.js conventions. Pages are discovered from the `app/` directory and mapped to URL patterns based on their file path.

## Directory Structure

```
app/
  layout.tsx          ← root layout (wraps all pages)
  page.tsx            ← /
  error.tsx           ← error boundary for /
  loading.tsx         ← suspense fallback for /
  not-found.tsx       ← 404 for /
  posts/
    page.tsx          ← /posts
    layout.tsx        ← wraps /posts and children
    [slug]/
      page.tsx        ← /posts/:slug
  (blog)/
    about/
      page.tsx        ← /about (group stripped from URL)
```

## Page Files

Each route is defined by a `page.{ext}` file. The extension depends on your renderer (e.g., `.tsx` for React, `.svelte` for Svelte, `.vue` for Vue). The page must have a default export.

## URL Patterns

| File Path | URL Pattern |
|---|---|
| `app/page.tsx` | `/` |
| `app/posts/page.tsx` | `/posts` |
| `app/posts/[slug]/page.tsx` | `/posts/:slug` |
| `app/posts/[...path]/page.tsx` | `/posts/:...path` (catch-all) |
| `app/posts/[[...path]]/page.tsx` | `/posts/:...path?` (optional catch-all) |
| `app/(shop)/products/page.tsx` | `/products` (route group stripped) |

## Dynamic Segments

Wrap a directory name in brackets to create a dynamic segment:

```
app/posts/[slug]/page.tsx    →  /posts/:slug
```

Access params in your component via the `params` prop:

```tsx
export default function PostPage({ params }: { params: { slug: string } }) {
    return <h1>Post: {params.slug}</h1>
}
```

### Catch-All Segments

```
app/docs/[...path]/page.tsx
```

Matches `/docs/a`, `/docs/a/b`, `/docs/a/b/c`, etc. The param is an array of path segments.

### Optional Catch-All Segments

```
app/docs/[[...path]]/page.tsx
```

Same as catch-all, but also matches the base `/docs` path.

## Route Groups

Wrap a directory name in parentheses to create a route group. Groups organize files without affecting the URL:

```
app/(marketing)/about/page.tsx   →  /about
app/(shop)/products/page.tsx     →  /products
```

## Companion Files

Each route segment can have companion files alongside `page.{ext}`:

| File | Purpose |
|---|---|
| `layout.{ext}` | Wraps the page and its children. Nested layouts compose. |
| `error.{ext}` | Error boundary for the segment. |
| `loading.{ext}` | Suspense fallback shown while the page loads. |
| `not-found.{ext}` | 404 page for the segment. |

### Global Companions

These live at the app root:

| File | Purpose |
|---|---|
| `global-error.{ext}` | Root-level error handler. |
| `global-not-found.{ext}` | Global 404 page (no matching route). |

## Export Naming

The framework derives export names from directory segments using CamelCase:

| File Path | Export Name |
|---|---|
| `app/page.tsx` | `Index` |
| `app/posts/page.tsx` | `Posts` |
| `app/posts/[slug]/page.tsx` | `PostsSlug` |
| `app/(shop)/products/page.tsx` | `Products` |

## Pattern Matching & Specificity

When multiple routes could match a URL, the most specific route wins. Specificity is determined per-segment:

1. **Static segments** — exact match (highest priority)
2. **Dynamic segments** — `:param`
3. **Catch-all segments** — `:...path`
4. **Optional catch-all** — `:...path?` (lowest priority)

At equal specificity, longer routes (more segments) win. Alphabetical order breaks ties.

## Client Components

Files containing `"use client"` as the first non-empty line are treated as client components. These are bundled separately and hydrated on the client.

## Coexistence with API Routes

Frontend pages and API routes share the same URL namespace. Pages always win for GET requests. API routes handle non-GET methods (POST, PUT, DELETE, etc.) and serve GET only when no matching page exists.

See [`routes/README.md`](../../routes/README.md) for API route documentation.

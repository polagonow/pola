# SSR Render Caching

## Overview

Pola implements a Next.js-style SSR caching strategy that eliminates redundant network requests on subsequent page visits. On the first visit, the client bootstraps from an HTML shell and streams Flight data via a second request. On subsequent visits, cached Flight data is embedded directly in the HTML shell — reducing page loads to a single request.

## Architecture

### Two-Request Model (First Visit)

```
Browser GET /posts
  ├── Request 1: HTML shell (no Content-Type: text/x-component)
  │   └── Server returns HTML with <script type="module"> loading Client.tsx
  └── Request 2: Flight data (Content-Type: text/x-component)
      └── Client.tsx fetches, React streams via createFromFetch()
```

On first visit, the `teeWriter` in `tryRendererServe` captures the Flight response bytes into the in-memory cache while streaming them to the client.

### Single-Request Model (Subsequent Visits)

```
Browser GET /posts
  └── Request 1: HTML shell with __POLA_SSR_DATA__ embedded
      └── Client.tsx finds the global, creates a synthetic Response,
          passes to createFromFetch() — no second fetch needed
```

On subsequent visits, the orchestrator's `handle()` method checks the cache before serving HTML. If cached Flight data exists, it is JSON-encoded and embedded as `self.__POLA_SSR_DATA__=...` in an inline `<script>` tag.

## Components

### teeWriter (`internal/orchestrator.go`)

Wraps `http.ResponseWriter` to capture all written bytes into a buffer while passing through to the real writer. After the renderer finishes streaming, the buffer is stored in the cache.

```go
type teeWriter struct {
    http.ResponseWriter
    buf []byte
}
```

### Cache Key Format

```
ssr:<url_path>?<raw_query>
```

Example: `ssr:/posts/hello-world?id=42`

### Default Cache

All code paths (pipeline, prebuilt, hot-reload) fall back to `memory.MustNew(0)` (1024-entry LRU) when no cache plugin is registered. This means caching works out of the box without explicit configuration.

### Cache TTL

Controlled by `Route.Revalidate` (`time.Duration`). When set to `0`, cached entries have no expiry. When set to a positive duration, entries expire after that period and the next request triggers a fresh render.

## Per-Request Memoization

The `MemoInjector` (`internal/memoinjector.go`) wraps runtime injectors with a JS Proxy that deduplicates Go bridge function calls within a single render. If a component calls `__DEPENDENCY_INJECTION__.GetPosts()` multiple times during one render, only the first call crosses the Go-JS boundary — subsequent calls return the cached result.

The memoization JS source lives in `engine/polyfill/polyfill.go` as `BridgeMemoSrc`, keeping it alongside other polyfills and maintaining renderer-agnosticism.

## Browser Cache Safety

The orchestrator sets `Vary: Content-Type` on all responses. This tells the browser's HTTP cache that the same URL may return different content depending on the request's `Content-Type` header, preventing it from serving cached Flight data when the user navigates back expecting an HTML page.

## Client Integration (`Client.tsx`)

```tsx
const ssrData = (self as typeof globalThis & { __POLA_SSR_DATA__?: string }).__POLA_SSR_DATA__;
const fetchPromise = ssrData
  ? Promise.resolve(new Response(ssrData, { headers: { "Content-Type": "text/x-component" } }))
  : fetch(location.pathname + location.search, { method: "GET", headers: { "Content-Type": "text/x-component" } });
```

When `__POLA_SSR_DATA__` is present, a synthetic `Response` is created from the embedded string — no network request occurs. The rest of the React rendering pipeline (`createFromFetch` → `render`/`hydrateRoot`) is identical in both paths.

## Configuration

| Route Field   | Type            | Description                                      |
|---------------|-----------------|--------------------------------------------------|
| `Revalidate`  | `time.Duration` | Cache TTL; `0` = no expiry                       |
| `Static`      | `bool`          | Marks route as fully static (future use)         |

To use a custom cache (e.g. Redis), register a `core.Cache` implementation via the plugin system. The default in-memory LRU will be bypassed automatically.

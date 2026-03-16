# GoJSX

A from-scratch Go SSR framework that implements the **React Server Components (RSC) Flight Protocol** — using [Goja](https://github.com/dop251/goja) as a pure-Go JS runtime and [esbuild](https://esbuild.github.io/) for bundling. No Node.js. No CGO. A single Go binary serves everything.

---

## What this is

GoJSX lets you write React Server Components in TSX that run inside a **Go process**. Go functions are injected directly into the JS runtime as globals or as a `ctx` object, so your components can call your database, cache, or any Go service directly — with no API layer in between.

The server streams the output using the **RSC Flight Protocol**: a line-delimited, chunk-based wire format that React's client runtime understands natively. Suspense boundaries resolve concurrently as Go goroutines and stream their content as they finish.

```tsx
// This runs in Go's Goja VM — not in Node.js
async function ProductList() {
  const products = await ctx.getProducts()   // ← calls a Go function directly
  return products.map(p => (
    <div key={p.id} className="product">
      <strong>{p.name}</strong> ${p.price}
    </div>
  ))
}
```

---

## How it works

### The request lifecycle

```
Browser
  │
  ▼  GET /products
Go net/http
  │
  ├─ matches Route → PropsFunc(r) builds props map
  ├─ acquires VM from VMPool
  ├─ injects __REQUEST__ context into the VM
  │
  ▼  Renderer.Render()
Goja VM
  │
  ├─ calls ProductsPage(props)            ← your TSX, compiled by esbuild
  ├─ tree walk: intrinsics → ReactNode
  │              functions → recurse
  │              promises  → SuspenseBoundary
  │
  ▼  FlightWriter  (streams over HTTP chunked transfer)
 0:J{"type":"div","props":{...}}          ← shell, sent immediately
 1:S{"id":1,"fallback":{...}}             ← suspense placeholder
  │
  ├─ goroutine per Suspense boundary
  │   each calls the async render fn
  │   streams result when done
  │
 1:J{"type":"section","props":{...}}      ← resolved boundary
  │
  ▼
Browser rsc-client.js
  ├─ parses Flight lines from <script type="text/x-component">
  ├─ builds real DOM nodes from the React node tree
  └─ swaps suspense placeholders as chunks arrive
```

### The Flight wire format

Every line is one chunk:

```
<id>:<type><json>\n
```

| Type | Meaning | Example |
|------|---------|---------|
| `J`  | React element tree (shell or resolved boundary) | `0:J{"type":"div","props":{...}}` |
| `S`  | Suspense placeholder | `1:S{"id":1,"fallback":{...}}` |
| `I`  | Client component module reference | `2:I{"id":"components/Counter","chunks":["/public/Counter-abc.js"]}` |
| `E`  | Error boundary payload | `3:E{"message":"db timeout"}` |
| `H`  | Resource hint (preload/prefetch) | `4:H{"href":"/font.woff2","as":"font"}` |

### The dual bundle

esbuild runs two separate builds at startup:

**Server bundle** (CommonJS, targeting Goja)
- Compiles all Server Component TSX files
- Marks client components (`"use client"`) as external — they become `ClientRef` objects in the tree, never executed server-side
- Runs once at startup; all VMs in the pool share the compiled `*goja.Program`

**Client bundle** (ESM, targeting browsers)
- Compiles only the `"use client"` components
- Code-splits with `Splitting: true` — shared dependencies land in `/public/chunks/`
- Produces a `manifest.json` mapping component IDs to their bundle filenames

### The VM pool

Goja VMs are expensive to create (they re-run the full program). The `VMPool` pre-warms VMs using `sync.Pool` so each request pays only the cost of calling one function, not re-parsing the bundle. After each request, per-request globals (`__REQUEST__`, `__renderResult__`) are deleted and the VM is returned to the pool.

---

## Project structure

```
gojsx/
│
├── main.go                    Entry point: boot, bundle, bridge, routes, HTTP
│
├── runtime/
│   ├── flight.go              RSC Flight Protocol encoder
│   │                          FlightWriter, ChunkType, ClientRef, ReactNode
│   │
│   ├── vm.go                  Goja VM pool and Go→JS bridge
│   │                          VMPool, BridgeConfig, GoFunc
│   │                          Globals (bare JS functions) + ctx object
│   │
│   ├── render.go              Server-side JSX renderer
│   │                          Renderer, RenderOptions
│   │                          walkNode → ReactNode tree
│   │                          ClientManifest, LoadManifest
│   │
│   ├── suspense.go            Concurrent Suspense scheduler
│   │                          SuspenseScheduler, SuspenseBoundary
│   │                          FlushAll → goroutines + channel merge
│   │                          PromiseToGo (sync promise resolution)
│   │
│   └── types.go               GoValue alias (goja.Value re-export)
│
├── build/
│   └── bundler.go             esbuild integration
│                              Bundle() → server CJS + client ESM + manifest
│                              InlineReactShim() → minimal React for Goja
│
├── app/
│   ├── pages/
│   │   └── index.tsx          Example page (Server Component)
│   │                          Uses ctx.getProducts() and ctx.getUser()
│   │
│   └── components/
│       ├── Counter.tsx         Client Component ("use client")
│       └── ThemeToggle.tsx     Client Component ("use client")
│
├── public/
│   ├── rsc-client.js          Browser Flight payload consumer
│   │                          Parses chunks, renders DOM, handles navigation
│   │
│   └── [generated]            esbuild output (Counter-[hash].js, etc.)
│                              manifest.json
│
├── vendor/                    All Go dependencies vendored (builds offline)
│   ├── github.com/dop251/goja
│   ├── github.com/evanw/esbuild
│   └── ...
│
└── go.mod
```

---

## Getting started

### Requirements

- Go 1.22 or later
- No Node.js required
- No CGO — pure Go

### Run

```bash
# Extract (if downloaded as archive)
tar -xzf gojsx-project.tar.gz
cd gojsx

# Build and run (uses vendored deps — no internet needed)
go run -mod=vendor main.go

# Open in browser
open http://localhost:3000
```

The server starts, runs esbuild to compile the TSX files, boots the VM pool, and begins serving.

### Build a binary

```bash
go build -mod=vendor -o gojsx .
./gojsx
```

---

## Core concepts

### Server Components

Any `.tsx` file without `"use client"` is a Server Component. It runs inside the Goja VM on every request and has access to all bridged Go functions.

```tsx
// app/products.tsx
// No "use client" → runs in Go's Goja VM

export function ProductsPage({ category }: { category?: string }) {
  const products = ctx.getProducts()           // Go function
  const user     = ctx.getUser(__REQUEST__.query.userId)  // per-request context

  return (
    <div className="page">
      <h1>Products</h1>
      {products.map(p => <ProductCard key={String(p.id)} product={p} />)}
    </div>
  )
}
```

### Client Components

Files with `"use client"` at the top are bundled by esbuild for the browser. The server never executes them — it only writes a `ClientRef` into the Flight stream, and the browser resolves it against `manifest.json`.

```tsx
"use client"
// app/components/AddToCart.tsx

import { useState } from "react"

export default function AddToCart({ productId }: { productId: number }) {
  const [added, setAdded] = useState(false)
  return (
    <button onClick={() => setAdded(true)}>
      {added ? "✓ Added" : "Add to cart"}
    </button>
  )
}
```

### Registering Go functions

In `main.go`, add functions to the bridge before creating the VM pool:

```go
bridge := runtime.BridgeConfig{
  // Globals — called as bare functions in any component
  Globals: map[string]runtime.GoFunc{
    "fetchJSON": func(args []runtime.GoValue) (any, error) {
      resp, _ := http.Get(args[0].String())
      defer resp.Body.Close()
      var v any
      json.NewDecoder(resp.Body).Decode(&v)
      return v, nil
    },
  },
  // Context — namespaced under ctx.fn() to avoid collisions
  Context: map[string]runtime.GoFunc{
    "getProducts": func(args []runtime.GoValue) (any, error) {
      return db.Query("SELECT * FROM products")  // real DB call
    },
    "getUser": func(args []runtime.GoValue) (any, error) {
      id := args[0].String()
      return userService.Find(id)
    },
  },
}
```

Inside any Server Component, these are available as:

```tsx
const data   = fetchJSON("https://api.example.com/data")  // global
const products = ctx.getProducts()                          // ctx object
const user     = ctx.getUser(__REQUEST__.query.id)          // with args
```

### Registering routes

Routes are registered manually in `main.go` after creating the app:

```go
app.Register(Route{
  Pattern: "/products",
  Export:  "ProductsPage",            // exported function name in the bundle
  PropsFunc: func(r *http.Request) map[string]any {
    return map[string]any{
      "category": r.URL.Query().Get("category"),
    }
  },
})
```

`Export` must match the name of the exported function in your TSX file.

### Per-request context

Every render injects a `__REQUEST__` global into the VM:

```tsx
// Available in any Server Component without any import
const userId = __REQUEST__.query.id     // URL query params
const path   = __REQUEST__.path         // /products
const method = __REQUEST__.method       // GET
const auth   = __REQUEST__.headers["Authorization"]
```

### Suspense and streaming

Wrap async data fetches in Suspense. The scheduler detects Promises in the tree, streams a placeholder immediately, then streams the resolved content as a goroutine completes.

```tsx
import { Suspense } from "react"

export function ProductsPage() {
  return (
    <div>
      <h1>Products</h1>
      <Suspense fallback={<p>Loading products…</p>}>
        <AsyncProductList />
      </Suspense>
    </div>
  )
}

async function AsyncProductList() {
  const products = await ctx.getProductsSlowQuery()  // returns a Promise
  return <div>{products.map(p => <ProductCard product={p} />)}</div>
}
```

The shell (`<h1>Products</h1>` + the loading fallback) streams to the browser in milliseconds. The product list streams as a second chunk when the Go goroutine finishes.

---

## HTTP endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Full HTML page — HTML shell + RSC payload inline |
| `GET /products` | Full HTML page for the products route |
| `GET /user?id=42` | Full HTML page for the user route |
| `GET /rsc?path=/` | Raw RSC Flight payload only (for client navigation) |
| `GET /public/*` | Static assets (client bundles, manifest) |

The `/rsc` endpoint returns `Content-Type: text/x-component` — the RSC MIME type. `rsc-client.js` uses this endpoint for client-side navigation (no full page reload).

---

## The React shim

Because Goja is not a browser and does not include React, `build/bundler.go` injects a minimal React shim before the server bundle runs. It implements:

- `React.createElement(type, props, ...children)`
- `React.Fragment`
- The automatic JSX runtime (`_jsx`, `_jsxs`, `_jsxDEV`, `Fragment`)

This is what makes `<div>Hello</div>` in your TSX work inside Goja without bundling the full `react` package into the server VM. For production, replace this with the actual `react-server` package from npm, bundled via esbuild's `Conditions: []string{"react-server"}` option.

---

## Adding a real database

Replace the mock `ctx.getProducts` in `main.go`:

```go
import "database/sql"
import _ "github.com/lib/pq"

db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))

bridge := runtime.BridgeConfig{
  Context: map[string]runtime.GoFunc{
    "getProducts": func(args []runtime.GoValue) (any, error) {
      rows, err := db.Query("SELECT id, name, price, stock FROM products ORDER BY name")
      if err != nil {
        return nil, err
      }
      defer rows.Close()
      var products []map[string]any
      for rows.Next() {
        var p struct { ID int; Name string; Price float64; Stock int }
        rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
        products = append(products, map[string]any{
          "id": p.ID, "name": p.Name, "price": p.Price, "stock": p.Stock,
        })
      }
      return products, rows.Err()
    },
  },
}
```

No changes needed in your TSX files — `ctx.getProducts()` just works.

---

## Known limitations and production roadmap

### Async/await in components

Goja does not run a Node.js event loop. The current Suspense integration handles **synchronously-resolved Promises** (i.e. Go functions that return immediately). For true `async/await` chains inside JSX, integrate [goja_nodejs](https://github.com/dop251/goja_nodejs) which provides a proper event loop:

```go
import "github.com/dop251/goja_nodejs/eventloop"

loop := eventloop.NewEventLoop()
loop.Start()
// Run each render inside loop.RunOnLoop(...)
```

### Client component hydration

The current `rsc-client.js` renders to real DOM nodes but does not hydrate client components with React. For full React hydration:

1. Include `react` and `react-dom` in your client bundle
2. Use `ReactDOM.createRoot(el).render(...)` in `rsc-client.js`
3. React's own RSC client (`react-server-dom-webpack`) handles the Flight payload natively

### Hot reload

esbuild runs once at startup. For development hot reload, add a file watcher (e.g. `fsnotify`) that re-runs `build.Bundle()` and calls `pool.Rebuild()` on source file changes.

### Multiple pages

Currently all pages are compiled into a single server bundle. For large apps, split into per-route bundles and use a separate `VMPool` per route, or use esbuild's `Splitting` option for the server build too.

---

## Dependencies

All dependencies are vendored in `vendor/` — the project builds with no internet access.

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/dop251/goja` | latest | Pure-Go JS/ES2020 runtime (no CGO) |
| `github.com/evanw/esbuild` | v0.20.2 | Go-native JS/TSX bundler |
| `github.com/dlclark/regexp2` | v1.11.4 | Goja dependency (regex engine) |
| `github.com/go-sourcemap/sourcemap` | v2.1.3 | Goja dependency (source maps) |
| `github.com/google/pprof` | latest | Goja dependency (profiling) |
| `golang.org/x/text` | v0.3.8 | Goja dependency (Unicode) |
| `golang.org/x/sys` | latest | esbuild dependency (OS syscalls) |

---

## Architecture decisions

**Why Goja over V8go?** Goja is pure Go — no CGO, no C++ toolchain, no shared libraries. It compiles to a single static binary that deploys anywhere Go runs. V8go is faster for CPU-heavy JS, but the CGO overhead and deployment complexity outweigh the gains for I/O-bound server rendering.

**Why implement Flight Protocol ourselves?** The official `react-server-dom-*` packages are Node.js-only. The protocol itself is straightforward enough to implement in ~150 lines of Go, and doing so means no Node.js subprocess, no IPC overhead, and full control over the streaming behaviour.

**Why esbuild as a Go package?** esbuild exposes its full API as an importable Go package (`github.com/evanw/esbuild/pkg/api`). This means the bundler is part of the Go binary — no separate build step, no `package.json`, no `node_modules`. The build runs in the same process at startup.

**Why manual route registration?** File-based routing (Next.js style) requires a filesystem watcher and dynamic code loading. Manual registration is explicit, testable, and pairs naturally with Go's idiomatic HTTP handler pattern. File-based routing can be layered on top by scanning a directory and calling `app.Register()` for each found page.

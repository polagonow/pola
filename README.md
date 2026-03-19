# Pola

A Go framework for **React Server Components (RSC)** — implements the Flight streaming protocol, Next.js-style file conventions, and a pluggable multi-VM architecture. No Node.js. No CGO by default. A single Go binary serves everything.

---

## What this is

Pola lets you write React Server Components in TSX that run inside a Go process. Go functions are exposed to the JS runtime via a typed bridge (`JSI`), so your components can call your database, cache, or any Go service directly — no API layer required.

The server streams output using the **RSC Flight Protocol**: React's native wire format. Suspense boundaries resolve concurrently and stream their content as they complete.

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

### The Flight wire format

Each line is one chunk: `<id>:<type><json>\n`

| Type | Meaning | Example |
|------|---------|---------|
| `J`  | React element tree | `0:["$","div",null,{"children":"Hello"}]` |
| `I`  | Client component module reference | `1:I{"id":"Counter","chunks":["counter-abc.js"]}` |
| `H`  | Resource hint (preload/prefetch) | `2:H{"href":"/font.woff2","as":"font"}` |
| `E`  | Error boundary payload | `3:E{"message":"db timeout"}` |
| `S`  | Suspense placeholder | `4:S{"id":1,"fallback":{...}}` |

### The dual bundle

esbuild runs two passes at startup:

**Client bundle** (ESM, browser)
- Compiles `"use client"` components and the client entry
- Code-split with shared chunks under `/public/assets/`
- Produces `manifest.json` mapping component IDs → chunk URLs

**Server bundle** (CJS, VM)
- Compiles all Server Component TSX with `react-server` condition
- `"use client"` files become `ClientRef` stubs — never executed server-side
- Loaded once; all VMs in the pool share the compiled program

### The VM pool

VMs are expensive to initialise (they parse and run the full server bundle). The pool pre-warms instances so each request pays only the cost of one function call. After each request, per-request state is cleared and the VM is returned to the pool.

---

## Project structure

```
pola/
│
├── framework/           Core interfaces + orchestration
│   ├── framework.go     Config.Build() pipeline, App, Handler
│   ├── interfaces.go    Pluggable interfaces (Bundler, VMFactory, Router, …)
│   ├── contract/        Shared data types — zero external deps
│   ├── hotreload/       fsnotify watcher + WebSocket dev server
│   ├── pubsub/          In-process broadcast bus
│   └── router/          Priority router (static > dynamic > catch-all)
│
├── vm/
│   ├── vm.go            Selects active VM via blank import
│   ├── goja/            Goja (pure-Go ES2020) — DEFAULT
│   ├── sobek/           Sobek (pure-Go, Goja fork)
│   ├── v8go/            V8 via CGO
│   ├── quickjsgo/       QuickJS via CGO
│   ├── moderncquickjs/  Modern QuickJS binding
│   └── qjs/             QuickJS via WASM (no CGO)
│
├── render/react/
│   ├── flight.go        Flight chunk types and writer
│   ├── protocol.go      RSCFlightProtocol (StreamProtocol impl)
│   └── discovery/nextjs/ NextJS-style discovery, entry generator, route builder
│
├── bundler/esbuild/     EsbuildBundler implementation
│
├── ui/apps/blog-e2e-react/    TypeScript source for the example blog
│   └── app/             Next.js-style app/ directory
│
├── example/blog/        Go entry point for the example
│   └── main.go
│
├── tests/               End-to-end test suite
├── mage.go              Zero-install mage bootstrap (go run mage.go <target>)
├── magefile.go          Build targets (mage)
├── .golangci.yml        golangci-lint configuration
├── lefthook.yml         Git hooks (pre-commit lint, pre-push test, commit-msg)
└── go.mod               Go 1.24, module: github.com/polagonow/pola
```

---

## Getting started

### Requirements

- Go 1.24 or later
- No Node.js required
- No CGO — pure Go (default Goja VM)

### Run the example

```bash
git clone <repo>
cd go-react-ssr-v2

go run mage.go run
# open http://localhost:3000
```

The server discovers pages, runs esbuild, boots the VM pool, and starts serving — all in one binary.

### Build a binary

```bash
go run mage.go build
./bin/pola
```

---

## File conventions (Next.js App Router)

Pages live under an `app/` directory. Pola discovers them automatically:

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

## Server Components

Any `.tsx` file without `"use client"` is a Server Component. It runs in the VM on every request.

```tsx
// app/posts/[slug]/page.tsx
import JSI from "@pola/jsi"

interface Props {
  params: { slug: string }
}

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

---

## Client Components

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

## Go↔JS bridge (JSI)

Go functions are exposed to Server Components via the `JSI` object. There are two scopes:

- **Globals** — bare functions callable anywhere: `fetchJSON(url)`, `getEnv(key)`
- **Context** — namespaced under `JSI.*`: `JSI.getPosts()`, `JSI.getPost(slug)`

### Defining the bridge

```go
import "github.com/polagonow/pola/framework/contract"

bridge := contract.BridgeConfig{
    Globals: map[string]contract.GoFunc{
        "getEnv": func(args []any) (any, error) {
            return os.Getenv(fmt.Sprintf("%v", args[0])), nil
        },
    },
    Context: map[string]contract.GoFunc{
        "getPosts": func(_ []any) (any, error) {
            return db.QueryPosts()
        },
        "getPost": func(args []any) (any, error) {
            slug := fmt.Sprintf("%v", args[0])
            return db.QueryPost(slug)
        },
    },
}
```

### Per-route bridges (least privilege)

Different routes can expose different Go functions:

```go
postsBridge := &contract.BridgeConfig{
    Context: map[string]contract.GoFunc{
        "getPosts": bridge.Context["getPosts"],
        "getPost":  bridge.Context["getPost"],
    },
}

// Apply after cfg.Build()
routeBridges := map[string]*contract.BridgeConfig{
    "/posts":        postsBridge,
    "/posts/:slug":  postsBridge,
    "/projects":     projectsBridge,
}
for i := range app.Artifacts().Routes {
    if b := routeBridges[app.Artifacts().Routes[i].Pattern]; b != nil {
        app.Artifacts().Routes[i].Bridge = b
    }
}
```

### Using JSI in components

```tsx
import JSI from "@pola/jsi"

const posts = await JSI.getPosts()           // Context function
const env   = getEnv("NEXT_PUBLIC_API_URL")  // Global function
```

---

## Starting the server

```go
package main

import (
    "net/http"
    "github.com/polagonow/pola/framework"
    "github.com/polagonow/pola/framework/contract"
    "github.com/polagonow/pola/framework/hotreload"
    _ "github.com/polagonow/pola/bundler/esbuild"          // register bundler
    _ "github.com/polagonow/pola/vm/goja"                  // register VM
    _ "github.com/polagonow/pola/render/react/discovery/nextjs" // register discovery + entry gen
)

func main() {
    cfg := &framework.Config{
        AppDir:       "./ui/apps/myapp",
        Dev:          true,
        GlobalBridge: bridge,
    }

    app, err := cfg.Build()
    if err != nil {
        log.Fatal(err)
    }

    // Apply per-route bridges
    for i := range app.Artifacts().Routes {
        if b := routeBridges[app.Artifacts().Routes[i].Pattern]; b != nil {
            app.Artifacts().Routes[i].Bridge = b
        }
    }

    handler := app.Handler()

    if cfg.Dev {
        reloader, err := hotreload.New(cfg, app, func(a *framework.App) {
            // Re-apply bridges after each hot reload
            for i := range a.Artifacts().Routes {
                if b := routeBridges[a.Artifacts().Routes[i].Pattern]; b != nil {
                    a.Artifacts().Routes[i].Bridge = b
                }
            }
        })
        if err != nil {
            log.Fatal(err)
        }
        defer reloader.Close()
        handler = reloader.Handler()
    }

    log.Fatal(http.ListenAndServe(":3000", handler))
}
```

---

## Hot reload

In development (`Dev: true`), Pola watches the `AppDir` for changes using `fsnotify`. On any `.tsx`/`.ts` file change it:

1. Re-runs discovery + bundling
2. Recompiles the VM pool
3. Sends a WebSocket message to all connected browsers → page reloads

The WebSocket endpoint is `/__dev__/hot`. The client script is injected automatically into the HTML shell when `Dev: true`.

---

## Selecting a JS VM

The active VM is chosen by a blank import in `vm/vm.go`. Swap it to change the engine:

```go
// vm/vm.go
import _ "github.com/polagonow/pola/vm/goja"           // Pure-Go ES2020 — default
// import _ "github.com/polagonow/pola/vm/sobek"        // Pure-Go Goja fork
// import _ "github.com/polagonow/pola/vm/qjs"          // QuickJS via WASM (no CGO)
// import _ "github.com/polagonow/pola/vm/quickjsgo"    // QuickJS via CGO
// import _ "github.com/polagonow/pola/vm/moderncquickjs"
// import _ "github.com/polagonow/pola/vm/v8go"         // V8 via CGO (fastest, needs C++ toolchain)
```

Each package registers itself via `init()` using `framework.RegisterDefaults(...)`. Only one VM can be active at a time.

| VM | CGO | Notes |
|----|-----|-------|
| `goja` | No | Default; ES2020, EventLoop, pure Go |
| `sobek` | No | Goja fork with improvements |
| `qjs` | No | QuickJS compiled to WASM |
| `quickjsgo` | Yes | QuickJS native binding |
| `moderncquickjs` | Yes | Modern QuickJS binding |
| `v8go` | Yes | V8 engine; fastest for CPU-heavy JS |

---

## Suspense and streaming

Wrap async data in `<Suspense>`. The shell streams immediately; resolved boundaries arrive as follow-up chunks.

```tsx
import { Suspense } from "react"
import JSI from "@pola/jsi"

export default function PostsPage() {
  return (
    <div>
      <h1>Posts</h1>
      <Suspense fallback={<p>Loading…</p>}>
        <PostList />
      </Suspense>
    </div>
  )
}

async function PostList() {
  const posts = await JSI.getPosts()  // slow query — runs in a goroutine
  return <ul>{posts.map(p => <li key={p.slug}>{p.title}</li>)}</ul>
}
```

The `<h1>` streams to the browser immediately. The `<ul>` streams as a second Flight chunk when the Go function returns.

---

## Development tooling

### Tasks (mage — zero install)

No `mage` binary needed. All targets run via `go run mage.go <target>`:

```bash
go run mage.go run             # Start dev server (example/blog)
go run mage.go build           # Compile binary → bin/pola
go run mage.go test            # Run all tests
go run mage.go testUnit        # Fast unit tests only
go run mage.go testE2E         # Full e2e suite (120s timeout)
go run mage.go testBuild       # Build/discover tests only
go run mage.go lint            # Run golangci-lint + eslint
go run mage.go uiLint          # ESLint across the UI monorepo
go run mage.go uiFormat        # Format UI files with prettier
go run mage.go uiFormatCheck   # Check UI formatting (no write)
go run mage.go installHooks    # Install git hooks via lefthook
go run mage.go clean           # Remove bin/ and public/assets/
go run mage.go -l              # List all targets
```

Environment variables can be set as usual before the command:

```bash
JS_VM=v8go CGO_ENABLED=1 go run mage.go run
```

### Linting

golangci-lint is configured in [.golangci.yml](.golangci.yml). Active linters: `govet`, `errcheck`, `staticcheck`, `gosimple`, `ineffassign`, `unused`, `misspell`, `revive`, `goimports`, `noctx`, `bodyclose`.

```bash
brew install golangci-lint
go run mage.go lint
```

### Git hooks (lefthook)

[lefthook.yml](lefthook.yml) configures three hooks:

| Hook | Command |
|------|---------|
| `pre-commit` | `golangci-lint run ./...` (on `.go` files) |
| `pre-push` | `go test -timeout 60s ./...` |
| `commit-msg` | Validates conventional commit format (`feat(scope): message`) |

```bash
brew install lefthook
go run mage.go installHooks
```

---

## Architecture decisions

**Why Goja over V8go by default?** Goja is pure Go — no CGO, no C++ toolchain, single static binary. V8go is faster for CPU-heavy JS but the CGO overhead and deployment complexity don't pay off for I/O-bound server rendering. The pluggable VM architecture means you can switch if you need to.

**Why implement the Flight Protocol ourselves?** The official `react-server-dom-*` packages are Node.js-only. The protocol is simple enough to implement in ~150 lines of Go, giving full control over streaming behaviour with no Node.js subprocess or IPC overhead.

**Why esbuild as a Go package?** esbuild exposes its full API as an importable Go package. The bundler is part of the binary — no separate build step, no `package.json`, no `node_modules`. It runs in the same process at startup.

**Why Next.js file conventions?** They're well-understood, tooled, and allow standard React codebases to target Pola with minimal changes. The discovery layer is an interface — you can replace it with your own convention.

**Why pluggable interfaces everywhere?** Swapping the VM, bundler, router, or renderer requires changing one blank import. The framework core has zero knowledge of any implementation.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/dop251/goja` | Pure-Go JS/ES2020 runtime |
| `github.com/dop251/goja_nodejs` | EventLoop for proper async/await in Goja |
| `github.com/evanw/esbuild` | Go-native JS/TSX bundler |
| `github.com/fsnotify/fsnotify` | Cross-platform file watcher (hot reload) |
| `github.com/gorilla/websocket` | WebSocket server (hot reload client push) |

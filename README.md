# GoJSX

Go + Goja + RSC Flight Protocol — server-render React Server Components inside Go using the Goja JS engine, with a real RSC Flight wire format pipeline and `react-server-dom-esm/client` on the browser.

## Architecture

```
Browser                         Go Server                    Goja VM
──────                          ─────────                    ───────
GET /           ────────────▶   htmlShell()                  (not involved)
                ◀────────────   <div id="root"> + <script>

createFromFetch(fetch('/rsc'))
GET /rsc?path=/ ────────────▶   handleRSC()
                                  vm = pool.Acquire()
                                  renderer.Render(fw, vm)  ──▶  __render__("IndexPage", "{}")
                                                                 renderToReadableStream(element, manifest)
                                                                 __pullStream__(stream) × N
                ◀────────────   0:["$","div",null,{...}]    ◀──  RSC Flight chunks
root.render(comp)
React tree rendered ✓
```

## How it works

**Server VM bundle** (Goja):
- Compiled with esbuild using the `react-server` condition
- Includes `react-server-dom-webpack/server.browser` for `renderToReadableStream`
- `renderToReadableStream(element, clientManifest)` → `ReadableStream` of RSC Flight chunks
- Go drains the stream via `__pullStream__` and writes chunks to `http.ResponseWriter`

**Browser bundle** (ESM):
- `client-entry.tsx` calls `createFromFetch(fetch('/rsc'))` using `react-server-dom-esm/client`
- `react-server-dom-esm` is a local build from `react-server-dom-turbopack` with `__turbopack_require__` replaced by native `import()` — the npm package (`0.0.1`) is an empty stub
- `createRoot(document.getElementById('root')).render(comp)` renders the React tree

**RSC Flight wire format** (`text/x-component`):
```
0:["$","div",null,{"className":"page","children":[...]}]
1:I{"id":"components/Counter","chunks":["/public/Counter-HASH.js"],"name":"default"}
```

## Polyfills (Goja)

`runtime/polyfills.js` provides the Web APIs `react-server-dom-webpack/server.browser` needs:
- `TextEncoder` / `TextDecoder`
- `ReadableStream` (with `__pullStream__` for Go to drain)
- `queueMicrotask` + `MessageChannel` (React's `scheduleWork` hooks)
- `AbortController` / `AbortSignal`
- `__webpack_require__` stub (backed by client manifest; only fires for async client modules)

## Project structure

```
gojsx/
├── app/
│   ├── client-entry.tsx        # Browser bootstrap — createFromFetch
│   ├── server-entry.tsx        # Goja entry — renderToReadableStream
│   ├── components/
│   │   ├── Counter.tsx         # "use client" — interactive counter
│   │   └── ThemeToggle.tsx     # "use client" — theme switcher
│   └── pages/
│       ├── index.tsx           # IndexPage server component
│       ├── products.tsx        # ProductsPage — ctx.getProducts()
│       └── user.tsx            # UserPage — ctx.getUser(id)
├── build/
│   └── bundler.go              # esbuild: 2-pass (client ESM + server CJS)
├── runtime/
│   ├── polyfills.js            # Web API polyfills for Goja
│   ├── vm.go                   # Goja VM pool
│   ├── render.go               # __render__ → __pullStream__ → Flight bytes
│   ├── flight.go               # FlightWriter (writes to http.ResponseWriter)
│   ├── walker.go               # Fallback manual tree walker
│   └── suspense.go             # SuspenseScheduler
├── main.go                     # HTTP server, routes, VM wiring
├── package.json
└── go.mod
```

## Running

```bash
npm install
go run .
# → http://localhost:3000
```

## Routes

| Path | Page | Data source |
|------|------|-------------|
| `/` | IndexPage | `ctx.getProducts()` |
| `/products` | ProductsPage | `ctx.getProducts()` |
| `/user?id=N` | UserPage | `ctx.getUser(id)` |

All routes served as RSC Flight from `/rsc?path=<route>`.
Client navigation: `<a>` clicks are intercepted (in a real app) and `createFromFetch` re-fetches.
# go-react-ssr-v2

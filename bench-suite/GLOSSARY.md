# Glossary

Short forms and terms used across this benchmark suite (`README.md`,
`RESULTS.md`, `FAIRNESS.md`, `CAPABILITIES.md`, `CHARTS.md`) — with what each
means *here* and a link to authoritative reference material.

## Rendering & React

| Term | Stands for | In this suite | Reference |
|---|---|---|---|
| **RSC** | React Server Components | Components that render on the server and stream to the client; Pola's core model. | [react.dev — Server Components](https://react.dev/reference/rsc/server-components) |
| **SSR** | Server-Side Rendering | Rendering React to HTML on the server (first response contains content). | [MDN — Server-side rendering](https://developer.mozilla.org/en-US/docs/Glossary/SSR) |
| **Flight** | React Flight (RSC wire protocol) | The chunked wire format RSC streams (`0:["$","main",…]`). | [react-server-dom-webpack](https://www.npmjs.com/package/react-server-dom-webpack) · [Flight spec notes](https://github.com/facebook/react/blob/main/packages/react-server-dom-webpack/README.md) |
| **RSDW** | `react-server-dom-webpack` | The React package that produces/consumes the Flight stream; Pola's `react` renderer and the `nodejs-rsc` baseline both use it. | [npm — react-server-dom-webpack](https://www.npmjs.com/package/react-server-dom-webpack) |
| **Suspense** | — | React boundary that streams a fallback until async content resolves (scenarios 2 & 4). | [react.dev — `<Suspense>`](https://react.dev/reference/react/Suspense) |
| **hydration** | — | Attaching interactivity to server-rendered HTML in the browser. | [react.dev — `hydrateRoot`](https://react.dev/reference/react-dom/client/hydrateRoot) |
| **`createRoot` / `hydrateRoot`** | — | Client render vs. hydration; Pola renders from Flight via `createRoot` rather than hydrating server HTML. | [`createRoot`](https://react.dev/reference/react-dom/client/createRoot) · [`hydrateRoot`](https://react.dev/reference/react-dom/client/hydrateRoot) |
| **`renderToPipeableStream`** | — | React's streaming SSR primitive. | [react.dev](https://react.dev/reference/react-dom/server/renderToPipeableStream) |
| **`"use client"` / island** | — | A Client Component boundary in an otherwise server tree (scenario 3). | [react.dev — `"use client"`](https://react.dev/reference/rsc/use-client) |
| **DOM** | Document Object Model | The rendered element tree; the conformance gate compares normalized DOM. | [MDN — DOM](https://developer.mozilla.org/en-US/docs/Web/API/Document_Object_Model) |

## Performance metrics & statistics

| Term | Stands for | In this suite | Reference |
|---|---|---|---|
| **TTFB** | Time To First Byte | Time from request start to the first response byte. | [MDN — TTFB](https://developer.mozilla.org/en-US/docs/Glossary/Time_to_first_byte) · [web.dev](https://web.dev/articles/ttfb) |
| **TTLB** | Time To Last Byte | Time to the final response byte (whole render streamed). | [web.dev — resource timing](https://web.dev/articles/custom-metrics#resource-timing-api) |
| **TTI** | Time To Interactive | Redefined per-entry as a browser-measured interactivity mark (RSC hydration has no like-for-like TTI). | [web.dev — TTI](https://web.dev/articles/tti) |
| **p50 / p95 / p99** | 50th / 95th / 99th percentile | Latency distribution points (median = p50). | [Wikipedia — Percentile](https://en.wikipedia.org/wiki/Percentile) |
| **CoV** | Coefficient of Variation | stddev ÷ mean (×100%) — run-to-run stability. | [Wikipedia — Coefficient of variation](https://en.wikipedia.org/wiki/Coefficient_of_variation) |
| **RSS** | Resident Set Size | Physical memory a process holds (idle and under load). | [Wikipedia — RSS](https://en.wikipedia.org/wiki/Resident_set_size) |
| **req/s** | requests per second | Throughput under load (autocannon). | [autocannon](https://github.com/mcollina/autocannon) |
| **gzip** | — | DEFLATE-based compression; payload sizes reported gzipped. | [MDN — Content-Encoding](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Encoding) · [RFC 1952](https://www.rfc-editor.org/rfc/rfc1952) |
| **brotli** | — | Google's compression algorithm; payload sizes also reported brotli'd. | [google/brotli](https://github.com/google/brotli) · [RFC 7932](https://www.rfc-editor.org/rfc/rfc7932) |

## JavaScript runtimes / engines

| Term | Stands for | In this suite | Reference |
|---|---|---|---|
| **VM** | Virtual Machine (JS engine) | The embedded JavaScript engine Pola runs server components in. | [Wikipedia — Virtual machine](https://en.wikipedia.org/wiki/Virtual_machine) |
| **JIT** | Just-In-Time compilation | Runtime machine-code compilation (V8 has it; goja/sobek don't). | [Wikipedia — JIT](https://en.wikipedia.org/wiki/Just-in-time_compilation) |
| **CGO** | C bindings for Go (`cgo`) | Required by V8/QuickJS engines; produces non-static binaries. | [pkg.go.dev — cgo](https://pkg.go.dev/cmd/cgo) |
| **WASM** | WebAssembly | fastschema/qjs runs QuickJS compiled to WASM (via wazero). | [webassembly.org](https://webassembly.org/) · [MDN — WebAssembly](https://developer.mozilla.org/en-US/docs/WebAssembly) |
| **V8** | — | Google's JS engine (Node.js, Chrome); the `v8go` variant. | [v8.dev](https://v8.dev/) |
| **QuickJS** | — | Fabrice Bellard's small JS engine; three Go bindings benched. | [bellard.org/quickjs](https://bellard.org/quickjs/) |
| **goja** | — | Pure-Go JS interpreter (Pola default). | [dop251/goja](https://github.com/dop251/goja) |
| **sobek** | — | Grafana's maintained goja fork. | [grafana/sobek](https://github.com/grafana/sobek) |
| **v8go** | — | V8 bindings for Go. | [rogchap.com/v8go](https://pkg.go.dev/rogchap.com/v8go) |
| **quickjsgo** | — | buke/quickjs-go (CGO QuickJS binding). | [buke/quickjs-go](https://github.com/buke/quickjs-go) |
| **qjs** | — | fastschema/qjs (WASM QuickJS binding). | [fastschema/qjs](https://github.com/fastschema/qjs) |
| **moderncquickjs** | — | modernc.org/quickjs (CGO-free QuickJS). | [modernc.org/quickjs](https://pkg.go.dev/modernc.org/quickjs) |
| **nativersc** | native RSC renderer | Pola renderer that serializes Flight in Go instead of via RSDW. | *(see `renderer/nativersc` in this repo)* |
| **wazero** | — | The zero-dependency WebAssembly runtime fastschema/qjs uses. | [wazero.io](https://wazero.io/) |
| **event loop / job queue** | — | The mechanism that runs promise continuations; async RSC needs it fully drained. | [MDN — Event loop](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model) |

## Web, protocol & framework

| Term | Stands for | In this suite | Reference |
|---|---|---|---|
| **HTTP** | HyperText Transfer Protocol | The transport; streaming uses chunked HTTP + flush-per-chunk. | [MDN — HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP) |
| **LRU** | Least Recently Used | Pola's default in-memory cache eviction policy. | [Wikipedia — Cache replacement (LRU)](https://en.wikipedia.org/wiki/Cache_replacement_policies#Least_recently_used_(LRU)) |
| **DI** | Dependency Injection | Pola's Go↔JS bridge (`__DEPENDENCY_INJECTION__`) exposing Go functions to JS. | [Wikipedia — Dependency injection](https://en.wikipedia.org/wiki/Dependency_injection) |
| **CSRF** | Cross-Site Request Forgery | On by default in Pola apps; adds a `<meta>` token to the shell. | [MDN — CSRF](https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/CSRF) · [OWASP](https://owasp.org/www-community/attacks/csrf) |
| **CSP** | Content Security Policy | Security-headers middleware injects a nonce into the shell. | [MDN — CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP) |
| **ORM** | Object-Relational Mapping | Pola's data layer (GORM/Ent) — not exercised by these scenarios. | [Wikipedia — ORM](https://en.wikipedia.org/wiki/Object%E2%80%93relational_mapping) |
| **CLI** | Command-Line Interface | The `pola` binary (`pola new` / `build` / `dev`). | [Wikipedia — CLI](https://en.wikipedia.org/wiki/Command-line_interface) |
| **N/A** | Not Applicable | A scenario an entry legitimately can't do (recorded, never approximated). | — |

## Units

| Term | Stands for | Reference |
|---|---|---|
| **ms / µs** | millisecond / microsecond | [SI prefixes](https://en.wikipedia.org/wiki/Metric_prefix) |
| **KiB / MiB / GiB** | kibibyte / mebibyte / gibibyte (1024-based) | [Wikipedia — Binary prefix](https://en.wikipedia.org/wiki/Binary_prefix) |
| **B** | byte | [Wikipedia — Byte](https://en.wikipedia.org/wiki/Byte) |

## Tooling & CI

| Term | Stands for | In this suite | Reference |
|---|---|---|---|
| **autocannon** | — | HTTP load-testing tool used for throughput/latency. | [mcollina/autocannon](https://github.com/mcollina/autocannon) |
| **Playwright / CDP** | — · Chrome DevTools Protocol | Browser-side measurement (DOM conformance, hydration). | [Playwright](https://playwright.dev/) · [CDP](https://chromedevtools.github.io/devtools-protocol/) |
| **esbuild** | — | The JS bundler (Pola default; also builds the Node baselines). | [esbuild.github.io](https://esbuild.github.io/) |
| **CI** | Continuous Integration | The GitHub Actions checks on the PR. | [Wikipedia — CI](https://en.wikipedia.org/wiki/Continuous_integration) |
| **CVE** | Common Vulnerabilities and Exposures | Trivy flags dependency CVEs. | [cve.org](https://www.cve.org/) |
| **Trivy** | — | The vulnerability scanner in CI. | [trivy.dev](https://trivy.dev/) |
| **golangci-lint** | — | The Go linter in CI (`only-new-issues` mode). | [golangci-lint.run](https://golangci-lint.run/) |

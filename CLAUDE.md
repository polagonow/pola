# GoJSX — Claude Code guide

## Project layout

```
bundler/          build-tag import files + esbuild implementation
framework/        core interfaces, Config, App
render/           build-tag import files + React implementation
vm/               build-tag import files + JS engine implementations
test/
  combo/          bundler×renderer combinations (e2e fixture combos)
  e2e/            end-to-end tests
  e2e/suite/      one file per test feature area
  fixture/        AppFixture, VMFixture, BundlerRendererFixture interfaces
  vm/             per-VM driver files (implement VMFixture)
  vm/polyfill/    polyfill tests (use ForEachVM)
ui/               TypeScript/React app monorepo
example/          runnable examples
magefile.go       build system (mage)
```

## Build tags

Every binary links only the implementations requested via build tags:

| Tag | Selects |
|-----|---------|
| `goja` `v8go` `sobek` `quickjsgo` `moderncquickjs` `qjs` | JS engine |
| `esbuild` | bundler |
| `react` | renderer |
| `embed` | embed pre-built assets; also **excludes** bundler code |

```
go build -tags "goja esbuild react"       ./...   # dev
go build -tags "embed v8go esbuild react" ./...   # release binary
```

## Test infrastructure (cross-product)

`fixture.ForEachApp` iterates every **VM × bundler+renderer** combo automatically.

- Register a new VM → `test/vm/<name>.go` + `vm/<name>_vm.go`
- Register a new combo → `test/combo/<bundler>_<renderer>.go`
- No other files need touching — tests run against every combo automatically.

## Skill guides

Use `/skill-name` to invoke, or Claude loads them automatically when relevant.

| Skill | Purpose |
|-------|---------|
| `/add-e2e-test` | Add an end-to-end test suite or test case |
| `/add-polyfill` | Add a JS Web API polyfill (`.js` + test) |
| `/add-vm` | Wire up a new JS engine |
| `/add-bundler` | Wire up a new bundler |
| `/add-renderer` | Wire up a new renderer (Vue, Svelte, …) |

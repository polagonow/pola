# test

E2E test infrastructure for the Pola framework.

Tests are parameterized over engine × renderer × bundler combos and run
automatically against every registered fixture.

## Structure

```
test/
  fixture/        ← AppFixture / PolyfillFixture interfaces + HTTP helpers
  combo/          ← registers engine+bundler+renderer combinations
  vm/             ← registers polyfill-only VM fixtures
  e2e/            ← test entry point + suites
    ssr_rendering_test.go   ← test entry (TestHTMLShell, TestServerComponentRendering, …)
    suite/                  ← one file per feature area
```

## Running tests

```bash
# Unit tests only (fast)
go test -tags "goja esbuild react nextjs" ./...

# E2E tests (builds the full app)
go test -tags "goja esbuild react nextjs" ./test/e2e/...

# Verbose, single suite
go test -tags "goja esbuild react nextjs" -v -run TestHTMLShell ./test/e2e/...
```

## Adding a new test suite

See `.claude/skills/add-e2e-test/SKILL.md`.

## HTTP helpers (`test/fixture`)

| Helper | What it does |
|--------|-------------|
| `fixture.RSC(t, f, path)` | GET with `Content-Type: text/x-component`; fails on non-200 |
| `fixture.RSCAny(t, f, path)` | RSC request, returns `(status, body)`, never fails |
| `fixture.Page(t, f, path)` | Normal HTML GET; fails on non-200 |
| `fixture.PageAny(t, f, path)` | HTML GET, returns `(status, body)`, never fails |
| `fixture.FlightTree(t, body)` | Parses root `0:` Flight row as JSON |
| `fixture.FlightContains(body, s)` | `strings.Contains` over the flight body |

## Fixture iteration

| Function | When to use |
|----------|-------------|
| `fixture.ForEachApp` | Test applies to all combos |
| `fixture.ForEachReactApp` | Test is React-specific (RSC / Flight output) |
| `fixture.ForEachVM` | Test runs against polyfill-only VM fixtures |

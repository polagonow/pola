# JS_VM selects the JavaScript engine: gojs (default), v8go, or quickjs
JS_VM ?= gojs

# JS_BUNDLER selects the JavaScript bundler: esbuild (default)
JS_BUNDLER ?= esbuild

# JS_RENDERER selects the renderer: react.js (default) or client
JS_RENDERER ?= react

# EMBED_ASSETS controls whether to embed UI assets in the Go binary (default: 1)
EMBED_ASSETS ?= 1

# CGO_ENABLED controls whether to enable CGO for V8Go builds (default: 1)
CGO_ENABLED ?= 1

.PHONY: run test test-e2e test-unit test-build lint ui-lint ui-format ui-format-check install-hooks build clean

## run: start the dev server (runs from example/example so relative paths resolve correctly)
run:
	cd example/blog && CGO_ENABLED=1 go run -tags "v8go esbuild react" .

## build: compile the Go binary
build:
	CGO_ENABLED=1 go build -tags "embed v8go esbuild react" -ldflags="-s -w" -o bin/gojsx ./...

## test: run all tests
test:
	go test ./...

## test-unit: run fast unit tests (skip e2e)
test-unit:
	go test ./build/... ./runtime/...

## test-e2e: run end-to-end tests across all VMs (builds bundles — slow)
test-e2e:
	go test -v -timeout 600s ./test/e2e/...

## test-build: run only build/discover tests
test-build:
	go test -v ./build/...

## lint: run golangci-lint (Go) and eslint (UI)
lint: ui-lint
	golangci-lint run ./...

## ui-lint: run eslint across the UI monorepo
ui-lint:
	cd ui && pnpm run lint

## ui-format: format UI source files with prettier
ui-format:
	cd ui && pnpm run format

## ui-format-check: check UI formatting without writing
ui-format-check:
	cd ui && pnpm run format:check

## install-hooks: install lefthook git hooks
install-hooks:
	lefthook install

## clean: remove compiled outputs
clean:
	rm -rf bin/ public/assets/

## help: list targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'

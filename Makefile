.PHONY: run test test-e2e test-unit test-build lint build clean

## run: start the dev server (runs from example/ so relative paths resolve correctly)
run:
	cd example && go run .

## build: compile the Go binary
build:
	go build -o bin/gojsx ./...

## test: run all tests
test:
	go test ./...

## test-unit: run fast unit tests (skip e2e)
test-unit:
	go test ./build/... ./runtime/...

## test-e2e: run end-to-end tests (builds bundles — slow)
test-e2e:
	go test -v -run 'Test' -timeout 120s ./tests/...

## test-build: run only build/discover tests
test-build:
	go test -v ./build/...

## lint: vet all packages
lint:
	go vet ./...

## clean: remove compiled outputs
clean:
	rm -rf bin/ public/assets/

## help: list targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'

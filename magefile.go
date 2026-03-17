//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// ---------------------------------------------------------------------------
// Configuration — mirror the Makefile variables
// ---------------------------------------------------------------------------

// jsVM selects the JavaScript engine: gojs (default), v8go, or quickjs
var jsVM = envOr("JS_VM", "gojs")

// jsBundler selects the JavaScript bundler: esbuild (default)
var jsBundler = envOr("JS_BUNDLER", "esbuild")

// jsRenderer selects the renderer: react (default)
var jsRenderer = envOr("JS_RENDERER", "react")

// embedAssets controls whether to embed UI assets in the Go binary (default: "1")
var embedAssets = envOr("EMBED_ASSETS", "1")

// cgoEnabled controls whether CGO is enabled (default: "1")
var cgoEnabled = envOr("CGO_ENABLED", "1")

// ---------------------------------------------------------------------------
// Tag helpers
// ---------------------------------------------------------------------------

// runtimeTags returns build tags for non-embed builds: "<jsVM> <jsBundler> <jsRenderer>"
func runtimeTags() string {
	return strings.Join([]string{jsVM, jsBundler, jsRenderer}, " ")
}

// buildTags returns build tags for binary builds, optionally prepending "embed".
func buildTags() string {
	tags := []string{}
	if embedAssets == "1" {
		tags = append(tags, "embed")
	}
	tags = append(tags, jsVM, jsBundler, jsRenderer)
	return strings.Join(tags, " ")
}

// ---------------------------------------------------------------------------
// Targets
// ---------------------------------------------------------------------------

// Run starts the dev server (runs from example/blog so relative paths resolve correctly).
func Run() error {
	fmt.Printf("→ JS_VM=%s JS_BUNDLER=%s JS_RENDERER=%s\n", jsVM, jsBundler, jsRenderer)
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": cgoEnabled},
		"go", "run",
		"-C", "example/blog",
		"-tags", runtimeTags(),
		".",
	)
}

// Build compiles the Go binary.
func Build() error {
	tags := buildTags()
	fmt.Printf("→ tags=%q CGO_ENABLED=%s\n", tags, cgoEnabled)
	if err := os.MkdirAll("bin", 0o755); err != nil {
		return err
	}
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": cgoEnabled},
		"go", "build",
		"-tags", tags,
		"-ldflags", "-s -w",
		"-o", "bin/gojsx",
		"./...",
	)
}

// Test runs all tests.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// TestUnit runs fast unit tests (skips e2e).
func TestUnit() error {
	return sh.RunV("go", "test", "./build/...", "./runtime/...")
}

// TestE2E runs end-to-end tests (builds bundles — slow).
func TestE2E() error {
	return sh.RunV("go", "test", "-v", "-run", "Test", "-timeout", "120s", "./test/...")
}

// TestBuild runs only build/discover tests.
func TestBuild() error {
	return sh.RunV("go", "test", "-v", "./build/...")
}

// Lint runs golangci-lint (Go) and eslint (UI).
func Lint() error {
	mg.Deps(UiLint)
	return sh.RunV("golangci-lint", "run", "./...")
}

// UiLint runs eslint across the UI monorepo.
func UiLint() error {
	return sh.RunV("pnpm", "--dir", "ui", "run", "lint")
}

// UiFormat formats UI source files with prettier.
func UiFormat() error {
	return sh.RunV("pnpm", "--dir", "ui", "run", "format")
}

// UiFormatCheck checks UI formatting without writing.
func UiFormatCheck() error {
	return sh.RunV("pnpm", "--dir", "ui", "run", "format:check")
}

// InstallHooks installs lefthook git hooks.
func InstallHooks() error {
	return sh.RunV("lefthook", "install")
}

// Clean removes compiled outputs.
func Clean() error {
	return sh.RunV("rm", "-rf", "bin/", "public/assets/")
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

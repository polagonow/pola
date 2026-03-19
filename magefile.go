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

var (
	polaVM       = envOr("POLA_VM", "goja")
	polaBundler  = envOr("POLA_BUNDLER", "esbuild")
	polaRenderer = envOr("POLA_RENDERER", "react")
	polaRouter   = envOr("POLA_ROUTER", "nextjs")
	polaCSS      = envOr("POLA_CSS", "none")
	embedAssets  = envOr("POLA_EMBED", "1")
	cgoEnabled   = envOr("CGO_ENABLED", "1")
	polaMetrics  = envOr("POLA_METRICS", "false")
	polaPprof    = envOr("POLA_PPROF", "false")
)

func runtimeTags() string {
	tags := []string{polaVM, polaBundler, polaRenderer, polaRouter}
	if polaCSS != "none" {
		tags = append(tags, polaCSS)
	}
	return strings.Join(tags, " ")
}

func buildTags() string {
	tags := []string{}
	if embedAssets == "1" {
		tags = append(tags, "embed")
	}
	tags = append(tags, polaVM, polaBundler, polaRenderer, polaRouter)
	if polaCSS != "none" {
		tags = append(tags, polaCSS)
	}
	return strings.Join(tags, " ")
}

// Run starts the dev server.
func Run() error {
	fmt.Printf("→ POLA_VM=%s POLA_BUNDLER=%s POLA_RENDERER=%s\n", polaVM, polaBundler, polaRenderer)
	return sh.RunWithV(
		map[string]string{
			"CGO_ENABLED":      cgoEnabled,
			"POLA_DEV":         "true",
			"POLA_METRICS":     polaMetrics,
			"POLA_PPROF":       polaPprof,
			"POLA_WEBAPP_PATH": "../../ui/apps/blog-e2e-react",
		},
		"go", "run",
		"-C", "cmd/server",
		"-tags", runtimeTags(),
		".",
	)
}

// Bundle pre-builds the JS/CSS assets into cmd/server/public/ so they can be
// embedded by the "embed" build tag. Called automatically by Build when
// POLA_EMBED=1 (the default).
func Bundle() error {
	fmt.Printf("→ bundling assets into cmd/server/public/\n")
	return sh.RunWithV(
		map[string]string{
			"CGO_ENABLED":      cgoEnabled,
			"POLA_BUILD_ONLY":  "true",
			"POLA_PUBLIC_DIR":  "./public",
			"POLA_WEBAPP_PATH": "../../ui/apps/blog-e2e-react",
		},
		"go", "run",
		"-C", "cmd/server",
		"-tags", runtimeTags(),
		".",
	)
}

// Build compiles the Go binary. When POLA_EMBED=1 (default) it first runs
// Bundle to generate assets, then bakes them into the binary with the embed tag.
func Build() error {
	if embedAssets == "1" {
		mg.Deps(Bundle)
	}
	tags := buildTags()
	fmt.Printf("→ tags=%q CGO_ENABLED=%s\n", tags, cgoEnabled)
	os.MkdirAll("bin", 0o755) //nolint:errcheck
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": cgoEnabled},
		"go", "build",
		"-tags", tags,
		"-ldflags", "-s -w",
		"-o", "bin/pola",
		"./cmd/server",
	)
}

// Test runs unit tests only (fast).
func Test() error {
	return sh.RunV("go", "test", "-tags", runtimeTags(), "./...")
}

// TestE2E runs end-to-end tests (slow, builds bundles).
func TestE2E() error {
	return sh.RunV("go", "test", "-v", "-tags", runtimeTags(), "-run", "Test", "-timeout", "120s", "./test/...")
}

// TestAll runs all tests against all registered combos.
func TestAll() error {
	mg.Deps(Test, TestE2E)
	return nil
}

// Benchmark runs performance benchmarks.
func Benchmark() error {
	return sh.RunV("go", "test", "-tags", runtimeTags(), "-bench=.", "-benchmem", "./benchmark/...")
}

// Lint runs golangci-lint and eslint.
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

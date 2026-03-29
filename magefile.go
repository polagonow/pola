//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	polaVM       = envOr("POLA_VM", "goja")
	polaBundler  = envOr("POLA_BUNDLER", "esbuild")
	polaRenderer = envOr("POLA_RENDERER", "react")
	polaRouter   = envOr("POLA_ROUTER", "nextjs")
	polaCSS      = envOr("POLA_CSS", "tailwind")
	cgoEnabled   = envOr("CGO_ENABLED", "1")
	polaMetrics  = envOr("POLA_METRICS", "false")
	polaPprof    = envOr("POLA_PPROF", "false")
)

// runtimeTags returns build tags for dev/bundle runs (includes bundler).
func runtimeTags() string {
	tags := []string{polaVM, polaBundler, polaRenderer, polaRouter}
	if polaCSS != "none" {
		tags = append(tags, polaCSS)
	}
	return strings.Join(tags, " ")
}

// Generate runs all code-generation steps (templ → Go).
// Run and Bundle depend on this so it always executes before compilation.
func Generate() error {
	fmt.Println("→ generating templ components")
	return sh.RunV("go", "tool", "templ", "generate", "./shell/...")
}

// Run starts the dev server for the blog-e2e-react example.
func RunDemo() error {
	mg.Deps(Generate, Build)
	fmt.Printf("→ POLA_VM=%s POLA_BUNDLER=%s POLA_RENDERER=%s\n", polaVM, polaBundler, polaRenderer)
	polaBin, _ := filepath.Abs("./bin/pola")
	if err := os.Chdir("examples/blog-e2e-react"); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}
	return sh.RunWithV(
		map[string]string{
			"CGO_ENABLED":  cgoEnabled,
			"POLA_DEV":     "true",
			"POLA_METRICS": polaMetrics,
			"POLA_PPROF":   polaPprof,
		},
		polaBin, "serve",
	)
}

// Bundle pre-builds JS/CSS assets for the blog-e2e-react example.
func BundleDemo() error {
	mg.Deps(Generate, Build)
	fmt.Println("→ bundling assets for examples/blog-e2e-react")
	polaBin, _ := filepath.Abs("./bin/pola")
	if err := os.Chdir("examples/blog-e2e-react"); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": cgoEnabled},
		polaBin, "build",
	)
}

// Build compiles the pola CLI binary into bin/.
func Build() error {
	mg.Deps(Generate)
	fmt.Println("→ building pola CLI")
	os.MkdirAll("bin", 0o755) //nolint:errcheck
	return sh.RunV("go", "build", "-o", "bin/pola", "./cmd/pola/")
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
	return sh.RunV("golangci-lint", "run", "./...")
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

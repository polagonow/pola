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

// Run starts the dev server for the blog-e2e-react example.
func RunDemo() error {
	mg.Deps(Build)
	fmt.Printf("→ POLA_VM=%s POLA_BUNDLER=%s POLA_RENDERER=%s\n", polaVM, polaBundler, polaRenderer)
	polaBin, _ := filepath.Abs("./bin/pola")
	if err := os.Chdir("examples/blog-e2e-react"); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}
	return sh.RunWithV(
		map[string]string{
			"CGO_ENABLED":      cgoEnabled,
			"POLA_ENV":         "development",
			"POLA_METRICS":     polaMetrics,
			"POLA_PPROF":       polaPprof,
			"POLA_WEBAPP_PATH": "../../examples/blog-e2e-react",
		},
		polaBin, "serve",
	)
}

// Bundle pre-builds JS/CSS assets for the blog-e2e-react example.
func BundleDemo() error {
	mg.Deps(Build)
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
	fmt.Println("→ building pola CLI")
	os.MkdirAll("bin", 0o755) //nolint:errcheck
	version := gitVersion()
	ldflags := fmt.Sprintf("-s -w -X github.com/polagonow/pola/internal/cli.version=%s", version)
	return sh.RunV("go", "build", "-trimpath", "-ldflags", ldflags, "-o", "bin/pola", "./cmd/pola/")
}

// Test runs unit tests only (fast) with race detection.
func Test() error {
	return sh.RunV("go", "test", "-race", "-tags", runtimeTags(), "./...")
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

// Cover runs tests with coverage profiling.
func Cover() error {
	return sh.RunV("go", "test", "-tags", runtimeTags(), "-coverprofile=coverage.out", "-covermode=atomic", "./...")
}

// Vet runs go vet with the active build tags.
func Vet() error {
	return sh.RunV("go", "vet", "-tags", runtimeTags(), "./...")
}

// Verify checks that module dependencies have not been tampered with.
func Verify() error {
	return sh.RunV("go", "mod", "verify")
}

// Tidy ensures go.mod and go.sum are up to date, and fails if they drift.
func Tidy() error {
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}
	return sh.RunV("git", "diff", "--exit-code", "go.mod", "go.sum")
}

func gitVersion() string {
	s, _ := sh.Output("git", "describe", "--tags", "--always", "--dirty")
	if s == "" {
		return "dev"
	}
	return s
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

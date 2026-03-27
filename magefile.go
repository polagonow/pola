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

// binaryTags returns build tags for the production binary.
// Excludes bundler and CSS plugins — both are build-time only; assets are
// pre-built by Bundle and embedded. Only the VM, renderer, and router are
// needed at runtime.
func binaryTags() string {
	return strings.Join([]string{"embed", polaVM, polaRenderer, polaRouter}, " ")
}

// Generate runs all code-generation steps (templ → Go).
// Run and Bundle depend on this so it always executes before compilation.
func Generate() error {
	fmt.Println("→ generating templ components")
	return sh.RunV("go", "tool", "templ", "generate", "./shell/...")
}

// Run starts the dev server.
func Run() error {
	mg.Deps(Generate)
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
		"-C", "cmd/blog-e2e-server",
		"-tags", runtimeTags(),
		".",
	)
}

// Bundle is Stage 1: pre-builds JS/CSS assets into cmd/blog-e2e-server/public/ using
// the full runtime tags (including esbuild). The server exits immediately after
// writing assets (POLA_BUILD_ONLY=true).
func Bundle() error {
	mg.Deps(Generate)
	fmt.Println("→ [stage 1] bundling assets into cmd/blog-e2e-server/public/")
	return sh.RunWithV(
		map[string]string{
			"CGO_ENABLED":      cgoEnabled,
			"POLA_BUILD_ONLY":  "true",
			"POLA_PUBLIC_DIR":  "./public",
			"POLA_WEBAPP_PATH": "../../ui/apps/blog-e2e-react",
		},
		"go", "run",
		"-C", "cmd/blog-e2e-server",
		"-tags", runtimeTags(),
		".",
	)
}

// Compile is Stage 2: compiles the production Go binary with the embed tag
// but WITHOUT esbuild — assets were already written by Bundle. mg.Deps ensures
// Bundle runs first (and is memoised so it never runs twice in one invocation).
func Compile() error {
	mg.Deps(Bundle)
	tags := binaryTags()
	fmt.Printf("→ [stage 2] compiling binary tags=%q CGO_ENABLED=%s\n", tags, cgoEnabled)
	os.MkdirAll("bin", 0o755) //nolint:errcheck
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": cgoEnabled},
		"go", "build",
		"-tags", tags,
		"-ldflags", "-s -w",
		"-o", "bin/blog-e2e-server",
		"./cmd/blog-e2e-server",
	)
}

// Build runs the full two-stage build: Bundle assets → Compile binary.
func Build() error {
	mg.Deps(Compile)
	return nil
}

// BuildCLI compiles the pola CLI binary into bin/.
func BuildCLI() error {
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

//go:build mage
// +build mage

package main

import (
	"archive/tar"
	"archive/zip"
	"cmp"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	polaVM       = cmp.Or(os.Getenv("POLA_VM"), "goja")
	polaBundler  = cmp.Or(os.Getenv("POLA_BUNDLER"), "esbuild")
	polaRenderer = cmp.Or(os.Getenv("POLA_RENDERER"), "react")
	polaRouter   = cmp.Or(os.Getenv("POLA_ROUTER"), "nextjs")
	polaCSS      = cmp.Or(os.Getenv("POLA_CSS"), "tailwind")
	cgoEnabled   = cmp.Or(os.Getenv("CGO_ENABLED"), "1")
	polaMetrics  = cmp.Or(os.Getenv("POLA_METRICS"), "false")
	polaPprof    = cmp.Or(os.Getenv("POLA_PPROF"), "false")
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
	return sh.RunV("go", "build", "-trimpath", "-ldflags", ldflags(), "-o", "bin/pola", "./cmd/pola/")
}

// BuildRace compiles the pola CLI binary into bin/ with the race detector enabled.
func BuildRace() error {
	fmt.Println("→ building pola CLI (race)")
	os.MkdirAll("bin", 0o755) //nolint:errcheck
	// The race detector requires cgo regardless of the POLA build configuration.
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": "1"},
		"go", "build", "-race", "-trimpath", "-ldflags", ldflags(), "-o", "bin/pola", "./cmd/pola/",
	)
}

// Install compiles and installs the pola CLI into GOBIN.
func Install() error {
	fmt.Println("→ installing pola CLI")
	return sh.RunV("go", "install", "-trimpath", "-ldflags", ldflags(), "./cmd/pola/")
}

// Uninstall removes the installed pola CLI from GOBIN.
func Uninstall() error {
	return sh.RunV("go", "clean", "-i", "./cmd/pola/")
}

// crossTarget is one GOOS/GOARCH the CLI builds for.
type crossTarget struct {
	goos, goarch string
}

// crossTargets is the platform matrix Pola cross-compiles to. Cross builds
// pin CGO off and the pure-Go goja VM (see crossTags), so the binary is fully
// static and portable; the CGO VM backends (v8go, quickjsgo) cannot be
// cross-compiled without a per-target C toolchain.
var crossTargets = []crossTarget{
	{goos: "darwin", goarch: "arm64"},
	{goos: "darwin", goarch: "amd64"},
	{goos: "linux", goarch: "arm64"},
	{goos: "linux", goarch: "amd64"},
	{goos: "windows", goarch: "arm64"},
	{goos: "windows", goarch: "amd64"},
}

// name is the flat release binary name in the standard Go convention,
// e.g. "pola-darwin-arm64" or "pola-windows-amd64.exe" — the same
// <base>-<GOOS>-<GOARCH> shape the CLI's own `pola build --platforms` emits
// (see internal/cli/platform.go) and that kubectl/minikube-style releases use.
func (t crossTarget) name() string {
	n := fmt.Sprintf("pola-%s-%s", t.goos, t.goarch)
	if t.goos == "windows" {
		n += ".exe"
	}
	return n
}

// binName is the binary name packaged inside release archives (.exe on
// Windows); archives carry the plain name so extraction yields `pola`.
func (t crossTarget) binName() string {
	if t.goos == "windows" {
		return "pola.exe"
	}
	return "pola"
}

// crossTags pins the pure-Go goja VM (required for CGO-free cross builds)
// while keeping the configured bundler/renderer/router/css — all pure Go.
func crossTags() string {
	tags := []string{"goja", polaBundler, polaRenderer, polaRouter}
	if polaCSS != "none" {
		tags = append(tags, polaCSS)
	}
	return strings.Join(tags, " ")
}

// buildTarget cross-compiles one platform into build/pola-<GOOS>-<GOARCH>[.exe].
func buildTarget(t crossTarget) error {
	out := filepath.Join("build", t.name())
	fmt.Printf("→ building %s\n", t.name())
	env := map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        t.goos,
		"GOARCH":      t.goarch,
	}
	return sh.RunWithV(env, "go", "build", "-trimpath", "-tags", crossTags(), "-ldflags", ldflags(), "-o", out, "./cmd/pola/")
}

// buildMany builds every target whose GOOS is in goos (empty = all targets).
func buildMany(goos ...string) error {
	want := make(map[string]bool, len(goos))
	for _, o := range goos {
		want[o] = true
	}
	for _, t := range crossTargets {
		if len(want) > 0 && !want[t.goos] {
			continue
		}
		if err := buildTarget(t); err != nil {
			return fmt.Errorf("build %s: %w", t.name(), err)
		}
	}
	return nil
}

// Dist groups the cross-platform build and release targets.
type Dist mg.Namespace

// All cross-compiles the pola CLI for every supported platform into build/.
func (Dist) All() error { return buildMany() }

// Linux cross-compiles the pola CLI for all supported Linux architectures.
func (Dist) Linux() error { return buildMany("linux") }

// Darwin cross-compiles the pola CLI for macOS (amd64 + arm64).
func (Dist) Darwin() error { return buildMany("darwin") }

// Windows cross-compiles the pola CLI for Windows (amd64 + arm64).
func (Dist) Windows() error { return buildMany("windows") }

// Release builds every platform and packages each binary into build/dist/ —
// pola-<version>-<GOOS>-<GOARCH>.tar.gz for Unix targets and .zip for
// Windows, each containing a plain pola[.exe]. Packaging is pure Go, so no
// external tar/zip tools are needed.
func (Dist) Release() error {
	if err := buildMany(); err != nil {
		return err
	}
	dist := filepath.Join("build", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return err
	}
	version := gitVersion()
	for _, t := range crossTargets {
		bin := filepath.Join("build", t.name())
		base := fmt.Sprintf("pola-%s-%s-%s", version, t.goos, t.goarch)
		var archive string
		var err error
		if t.goos == "windows" {
			archive = filepath.Join(dist, base+".zip")
			err = zipBinary(archive, bin, t.binName())
		} else {
			archive = filepath.Join(dist, base+".tar.gz")
			err = tarGzBinary(archive, bin, t.binName())
		}
		if err != nil {
			return fmt.Errorf("package %s: %w", filepath.Base(archive), err)
		}
		fmt.Printf("→ packaged %s\n", filepath.Base(archive))
	}
	return nil
}

// tarGzBinary writes a .tar.gz archive containing src as name (mode 0755).
func tarGzBinary(archive, src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: name, Mode: 0o755, Size: info.Size(), ModTime: info.ModTime()}
	if err := tw.WriteHeader(hdr); err != nil {
		out.Close() //nolint:errcheck
		return err
	}
	if _, err := io.Copy(tw, in); err != nil {
		out.Close() //nolint:errcheck
		return err
	}
	for _, c := range []io.Closer{tw, gw, out} {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}

// zipBinary writes a .zip archive containing src as name (mode 0755).
func zipBinary(archive, src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	hdr := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: info.ModTime()}
	hdr.SetMode(0o755)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		out.Close() //nolint:errcheck
		return err
	}
	if _, err := io.Copy(w, in); err != nil {
		out.Close() //nolint:errcheck
		return err
	}
	for _, c := range []io.Closer{zw, out} {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
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

// Fmt formats Go code in place using golangci-lint's gofmt + goimports formatters.
func Fmt() error {
	return sh.RunV("golangci-lint", "fmt", "./...")
}

// Check runs the full pre-flight gate: lint, vet, then the race-enabled test suite.
func Check() error {
	mg.Deps(Lint, Vet)
	return Test()
}

// InstallHooks installs lefthook git hooks.
func InstallHooks() error {
	return sh.RunV("lefthook", "install")
}

// Clean removes compiled outputs.
func Clean() error {
	return sh.RunV("rm", "-rf", "bin/", "build/", "public/assets/", "coverage.out")
}

// CleanTest clears the Go test cache.
func CleanTest() error {
	return sh.RunV("go", "clean", "-testcache")
}

// Cover runs tests with coverage profiling.
func Cover() error {
	return sh.RunV("go", "test", "-tags", runtimeTags(), "-coverprofile=coverage.out", "-covermode=atomic", "./...")
}

// CoverHTML runs coverage profiling and opens the HTML report.
func CoverHTML() error {
	mg.Deps(Cover)
	return sh.RunV("go", "tool", "cover", "-html=coverage.out")
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

// ldflags returns the linker flags that strip debug info and inject the CLI version.
func ldflags() string {
	return fmt.Sprintf("-s -w -X github.com/polagonow/pola/internal/cli.version=%s", gitVersion())
}

func gitVersion() string {
	s, _ := sh.Output("git", "describe", "--tags", "--always", "--dirty")
	if s == "" {
		return "dev"
	}
	return s
}

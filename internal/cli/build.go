package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/polagonow/pola/internal/cli/buildtags"
	"github.com/polagonow/pola/internal/cli/stubpkgs"
	"github.com/spf13/cobra"
)

var buildFlags struct {
	output   string
	renderer string
	bundler  string
	router   string
	css      string
	vm       string
	cgo      string
	appPath  string
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a production binary",
	Long: `Build a production-ready single binary in two stages:
  1. Bundle: pre-build JS/CSS assets using the full runtime
  2. Compile: compile a Go binary with embedded assets`,
	RunE: runBuild,
	Example: `  pola build
  pola build -o ./bin/myapp
  pola build --vm goja --renderer react`,
}

func init() {
	buildCmd.Flags().StringVarP(&buildFlags.output, "output", "o", "", "output binary path (default: ./bin/<app-name>)")
	buildCmd.Flags().StringVar(&buildFlags.renderer, "renderer", envOr("POLA_RENDERER", "react"), "view renderer")
	buildCmd.Flags().StringVar(&buildFlags.bundler, "bundler", envOr("POLA_BUNDLER", "esbuild"), "JS bundler")
	buildCmd.Flags().StringVar(&buildFlags.router, "router", envOr("POLA_ROUTER", "nextjs"), "router style")
	buildCmd.Flags().StringVar(&buildFlags.css, "css", envOr("POLA_CSS", "tailwind"), "CSS processor")
	buildCmd.Flags().StringVar(&buildFlags.vm, "vm", envOr("POLA_VM", "goja"), "JS engine")
	buildCmd.Flags().StringVar(&buildFlags.cgo, "cgo", envOr("CGO_ENABLED", "1"), "CGO_ENABLED value")
	buildCmd.Flags().StringVar(&buildFlags.appPath, "app-path", envOr("POLA_WEBAPP_PATH", "./app"), "path to the web app directory")
}

func runBuild(_ *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	// Determine output path.
	output := buildFlags.output
	if output == "" {
		appName := filepath.Base(projectDir)
		output = filepath.Join("bin", appName)
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(projectDir, output)
	}

	// Stub @pola/di and @pola/react into node_modules.
	if err := stubpkgs.StubToNodeModules(projectDir); err != nil {
		return fmt.Errorf("stub packages: %w", err)
	}

	// Verify embed.go exists.
	embedPath := filepath.Join(projectDir, "embed.go")
	if _, err := os.Stat(embedPath); os.IsNotExist(err) {
		return fmt.Errorf("embed.go not found in %s — required for production builds", projectDir)
	}

	// Generate overlay (plugin imports + action bridge codegen).
	overlayRes, err := generateOverlay(projectDir, buildFlags.css)
	if err != nil {
		return err
	}
	if overlayRes != nil && overlayRes.TmpDir != "" {
		defer os.RemoveAll(overlayRes.TmpDir)
	}

	// Run templ generate if templ files exist.
	if hasTemplFiles(projectDir) {
		fmt.Println("Generating templ components...")
		if err := runInDir(projectDir, "go", "tool", "templ", "generate", "./shell/..."); err != nil {
			if verbose {
				fmt.Printf("Warning: templ generate failed: %v\n", err)
			}
		}
	}

	// Collect overlay args.
	var overlayArgs []string
	if overlayRes != nil && overlayRes.OverlayPath != "" {
		overlayArgs = []string{"-overlay", overlayRes.OverlayPath}
	}

	// ── Stage 1: Bundle ──────────────────────────────────────────────────
	runtimeTags := buildtags.RuntimeTags(buildFlags.vm, buildFlags.bundler, buildFlags.renderer, buildFlags.router, buildFlags.css)
	fmt.Printf("[stage 1] Bundling assets (tags: %s)...\n", runtimeTags)

	bundleArgs := []string{"run"}
	bundleArgs = append(bundleArgs, overlayArgs...)
	bundleArgs = append(bundleArgs, "-tags", runtimeTags, ".")

	bundleCmd := exec.Command("go", bundleArgs...)
	bundleCmd.Dir = projectDir
	bundleCmd.Stdout = os.Stdout
	bundleCmd.Stderr = os.Stderr
	bundleCmd.Env = append(os.Environ(),
		"CGO_ENABLED="+buildFlags.cgo,
		"POLA_BUILD_ONLY=true",
		"POLA_PUBLIC_DIR=./public",
		"POLA_WEBAPP_PATH="+buildFlags.appPath,
	)

	if err := bundleCmd.Run(); err != nil {
		return fmt.Errorf("bundle stage failed: %w", err)
	}

	// ── Stage 2: Compile ─────────────────────────────────────────────────
	binaryTags := buildtags.BinaryTags(buildFlags.vm, buildFlags.renderer, buildFlags.router)
	fmt.Printf("[stage 2] Compiling binary (tags: %s)...\n", binaryTags)

	// Ensure output directory exists.
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	compileArgs := []string{"build"}
	compileArgs = append(compileArgs, overlayArgs...)
	compileArgs = append(compileArgs, "-tags", binaryTags, "-ldflags", "-s -w", "-o", output, ".")

	compileCmd := exec.Command("go", compileArgs...)
	compileCmd.Dir = projectDir
	compileCmd.Stdout = os.Stdout
	compileCmd.Stderr = os.Stderr
	compileCmd.Env = append(os.Environ(),
		"CGO_ENABLED="+buildFlags.cgo,
	)

	if err := compileCmd.Run(); err != nil {
		return fmt.Errorf("compile stage failed: %w", err)
	}

	fmt.Printf("Built %s\n", output)
	return nil
}

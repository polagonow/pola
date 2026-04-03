package cli

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/polagonow/pola/internal/autoload"
	"github.com/polagonow/pola/internal/stubpkgs"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

var buildFlags struct {
	output          string
	renderer        string
	bundler         string
	router          string
	css             string
	vm              string
	cgo             string
	appPath         string
	csrf            bool
	securityHeaders bool
	imageProcessing string
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
	buildCmd.Flags().StringVar(&buildFlags.renderer, "renderer", cmp.Or(os.Getenv("POLA_RENDERER"), "react"), "view renderer")
	buildCmd.Flags().StringVar(&buildFlags.bundler, "bundler", cmp.Or(os.Getenv("POLA_BUNDLER"), "esbuild"), "JS bundler")
	buildCmd.Flags().StringVar(&buildFlags.router, "router", cmp.Or(os.Getenv("POLA_ROUTER"), "nextjs"), "router style")
	buildCmd.Flags().StringVar(&buildFlags.css, "css", cmp.Or(os.Getenv("POLA_CSS"), "tailwind"), "CSS processor")
	buildCmd.Flags().StringVar(&buildFlags.vm, "vm", cmp.Or(os.Getenv("POLA_VM"), "goja"), "JS engine")
	buildCmd.Flags().StringVar(&buildFlags.cgo, "cgo", cmp.Or(os.Getenv("CGO_ENABLED"), "1"), "CGO_ENABLED value")
	buildCmd.Flags().StringVar(&buildFlags.appPath, "app-path", cmp.Or(os.Getenv("POLA_WEBAPP_PATH"), "./web"), "path to the web app directory")
	buildCmd.Flags().BoolVar(&buildFlags.csrf, "csrf", os.Getenv("POLA_CSRF") != "false", "enable CSRF protection")
	buildCmd.Flags().BoolVar(&buildFlags.securityHeaders, "security-headers", os.Getenv("POLA_SECURITY_HEADERS") != "false", "enable security headers")
	buildCmd.Flags().StringVar(&buildFlags.imageProcessing, "image-processing", os.Getenv("POLA_IMAGE_PROCESSING"), "image processing adapter")
}

func runBuild(cmd *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	applyPolafileDefaults(cmd, projectDir)

	// Load Polafile for PolaPackage.
	var pf polafile.Polafile
	pfPtr, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pfPtr != nil {
		pf = *pfPtr
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

	// Stub @pola/actions and @pola/react into node_modules.
	if err := stubpkgs.StubToNodeModules(filepath.Join(projectDir, buildFlags.appPath)); err != nil {
		return fmt.Errorf("stub packages: %w", err)
	}

	baseOpts := autoload.PluginOpts{
		PolaPackage:     pf.PolaPackage(),
		Engine:          buildFlags.vm,
		Bundler:         buildFlags.bundler,
		Renderer:        buildFlags.renderer,
		Router:          buildFlags.router,
		CSS:             buildFlags.css,
		Cache:           "memory",
		CSRF:            buildFlags.csrf,
		SecurityHeaders: buildFlags.securityHeaders,
		AppDir:          pf.AppDir(),
		ActionsDir:      generateFlags.actionsDir,
		TSOut:           generateFlags.tsOut,
		ImageProcessing: buildFlags.imageProcessing,
	}
	autoload.PopulateDatabaseOpts(&baseOpts, &pf, "production")
	autoload.PopulateStorageOpts(&baseOpts, &pf, "production")
	autoload.ApplyMailerOpts(&baseOpts, &pf, "production")

	// ── Stage 1: Bundle ──────────────────────────────────────────────────
	// Full runtime with bundler, osfs, css — needed to produce assets.
	fmt.Println("[stage 1] Bundling assets...")

	stage1Opts := baseOpts
	stage1Opts.Dev = false
	stage1Opts.Embed = false

	stage1Result, err := autoload.Run(projectDir, stage1Opts, verbose)
	if err != nil {
		return err
	}
	defer cleanupOverlay(stage1Result)

	bundleArgs := []string{"run"}
	if stage1Result != nil && stage1Result.OverlayPath != "" {
		bundleArgs = append(bundleArgs, "-overlay", stage1Result.OverlayPath)
	}
	bundleArgs = append(bundleArgs, ".")

	bundleCmd := exec.Command("go", bundleArgs...)
	bundleCmd.Dir = projectDir
	bundleCmd.Stdout = os.Stdout
	bundleCmd.Stderr = os.Stderr
	bundleCmd.Env = append(os.Environ(),
		"CGO_ENABLED="+buildFlags.cgo,
		"POLA_BUILD_ONLY=true",
		"POLA_PUBLIC_DIR=./public",
		"POLA_WEBAPP_PATH="+buildFlags.appPath,
		"GONOSUMCHECK=*",
		"GONOSUMDB=*",
		"GOFLAGS=-mod=mod",
	)

	if err := bundleCmd.Run(); err != nil {
		return fmt.Errorf("bundle stage failed: %w", err)
	}

	// ── Stage 2: Compile ─────────────────────────────────────────────────
	// Embed mode: no bundler/osfs/css, assets embedded via //go:embed.
	fmt.Println("[stage 2] Compiling binary...")

	stage2Opts := baseOpts
	stage2Opts.Dev = false
	stage2Opts.Embed = true

	stage2Result, err := autoload.Run(projectDir, stage2Opts, verbose)
	if err != nil {
		return err
	}
	defer cleanupOverlay(stage2Result)

	// Ensure output directory exists.
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	compileArgs := []string{"build"}
	if stage2Result != nil && stage2Result.OverlayPath != "" {
		compileArgs = append(compileArgs, "-overlay", stage2Result.OverlayPath)
	}
	compileArgs = append(compileArgs, "-ldflags", "-s -w", "-o", output, ".")

	compileCmd := exec.Command("go", compileArgs...)
	compileCmd.Dir = projectDir
	compileCmd.Stdout = os.Stdout
	compileCmd.Stderr = os.Stderr
	compileCmd.Env = append(os.Environ(),
		"CGO_ENABLED="+buildFlags.cgo,
		"GONOSUMCHECK=*",
		"GONOSUMDB=*",
		"GOFLAGS=-mod=mod",
	)

	if err := compileCmd.Run(); err != nil {
		return fmt.Errorf("compile stage failed: %w", err)
	}

	fmt.Printf("Built %s\n", output)
	return nil
}

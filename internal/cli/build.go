package cli

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	platforms       []string
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a production binary",
	Long: `Build a production-ready single binary in two stages:
  1. Bundle: pre-build JS/CSS assets using the full runtime
  2. Compile: compile a Go binary with embedded assets

Cross-compile for other platforms with --platform:
  pola build --platform linux/amd64
  pola build --platform all
  pola build --platform linux --platform darwin`,
	RunE: runBuild,
	Example: `  pola build
  pola build -o ./bin/myapp
  pola build --vm goja --renderer react
  pola build --platform linux/amd64
  pola build --platform all -o ./dist/myapp`,
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
	buildCmd.Flags().StringArrayVar(&buildFlags.platforms, "platform", nil, `target platform(s) as GOOS/GOARCH (e.g. linux/amd64); repeatable; use "all" for all supported`)
}

func runBuild(cmd *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	applyPolafileDefaults(cmd, projectDir)

	// Parse --platform targets.
	targets, err := parsePlatforms(buildFlags.platforms)
	if err != nil {
		return err
	}
	multiTarget := len(targets) > 1
	if len(targets) == 0 {
		targets = []platformTarget{{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}}
	}

	hasCross := false
	for _, t := range targets {
		if !t.isHost() {
			hasCross = true
			break
		}
	}
	if hasCross {
		if cmd.Flags().Changed("cgo") && buildFlags.cgo != "0" {
			fmt.Printf("Warning: cross-compilation forces CGO_ENABLED=0 (overriding --cgo=%s)\n", buildFlags.cgo)
		}
		if cmd.Flags().Changed("vm") && buildFlags.vm != "goja" {
			fmt.Printf("Warning: cross-compilation forces VM=goja (overriding --vm=%s)\n", buildFlags.vm)
		}
	}

	// Load Polafile for PolaPackage.
	var pf polafile.Polafile
	pfPtr, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	pfPtr = polafile.ApplyEnvOverrides(pfPtr)
	if pfPtr != nil {
		pf = *pfPtr
	}

	isAPIOnly := pfPtr != nil && pfPtr.IsAPIOnly()

	// Determine output path.
	output := buildFlags.output
	if output == "" {
		appName := filepath.Base(projectDir)
		output = filepath.Join("bin", appName)
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(projectDir, output)
	}

	baseOpts := autoload.PluginOpts{
		PolaPackage:     pf.PolaPackage(),
		Cache:           "memory",
		CSRF:            buildFlags.csrf,
		SecurityHeaders: buildFlags.securityHeaders,
		ActionsDir:      generateFlags.actionsDir,
		TSOut:           generateFlags.tsOut,
		ImageProcessing: buildFlags.imageProcessing,
		APIOnly:         isAPIOnly,
	}
	if !isAPIOnly {
		baseOpts.Engine = buildFlags.vm
		baseOpts.Bundler = buildFlags.bundler
		baseOpts.Renderer = buildFlags.renderer
		baseOpts.Router = buildFlags.router
		baseOpts.CSS = buildFlags.css
		baseOpts.AppDir = pf.AppDir()
	}

	defaultEnv := "production"
	autoload.PopulateDatabaseOpts(&baseOpts, &pf, defaultEnv)
	autoload.PopulateStorageOpts(&baseOpts, &pf, defaultEnv)
	autoload.ApplyMailerOpts(&baseOpts, &pf, defaultEnv)
	autoload.PopulateMCPOpts(&baseOpts, &pf, defaultEnv)
	autoload.PopulateSessionOpts(&baseOpts, &pf, defaultEnv)
	autoload.PopulateRateLimitOpts(&baseOpts, &pf, defaultEnv)
	autoload.PopulateFlashOpts(&baseOpts, &pf, defaultEnv)
	autoload.PopulateI18nOpts(&baseOpts, &pf, defaultEnv)

	if !isAPIOnly {
		// Auto-install frontend dependencies if not yet installed.
		webDir := filepath.Join(projectDir, buildFlags.appPath)
		if err := ensureFrontendDeps(webDir, pf); err != nil {
			return err
		}

		// Stub @pola/actions and @pola/react into node_modules.
		if err := stubpkgs.StubToNodeModules(filepath.Join(projectDir, buildFlags.appPath)); err != nil {
			return fmt.Errorf("stub packages: %w", err)
		}

		// ── Stage 1: Bundle ──────────────────────────────────────────────────
		// Full runtime with bundler, osfs, css — needed to produce assets.
		// Assets are platform-independent, so this runs once for the host.
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
	}

	// ── Compile ──────────────────────────────────────────────────────────
	// For cross targets, force goja VM so the overlay uses the pure-Go engine.
	stage2Opts := baseOpts
	stage2Opts.Dev = false
	stage2Opts.Embed = true
	if hasCross {
		stage2Opts.Engine = "goja"
	}

	stage2Result, err := autoload.Run(projectDir, stage2Opts, verbose)
	if err != nil {
		return err
	}
	defer cleanupOverlay(stage2Result)

	for i, target := range targets {
		if multiTarget {
			fmt.Printf("[stage 2] Compiling %s (%d/%d)...\n", target.label(), i+1, len(targets))
		} else if isAPIOnly {
			fmt.Println("Compiling binary...")
		} else {
			fmt.Println("[stage 2] Compiling binary...")
		}

		outputPath := resolveOutputPath(output, target, multiTarget)

		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("create output dir for %s: %w", target.label(), err)
		}

		cgoVal := buildFlags.cgo
		if !target.isHost() {
			cgoVal = "0"
		}

		compileArgs := []string{"build"}
		if stage2Result != nil && stage2Result.OverlayPath != "" {
			compileArgs = append(compileArgs, "-overlay", stage2Result.OverlayPath)
		}
		compileArgs = append(compileArgs, "-trimpath", "-ldflags", "-s -w", "-o", outputPath, ".")

		compileCmd := exec.Command("go", compileArgs...)
		compileCmd.Dir = projectDir
		compileCmd.Stdout = os.Stdout
		compileCmd.Stderr = os.Stderr
		compileCmd.Env = append(os.Environ(),
			"CGO_ENABLED="+cgoVal,
			"GOOS="+target.GOOS,
			"GOARCH="+target.GOARCH,
			"GONOSUMCHECK=*",
			"GONOSUMDB=*",
			"GOFLAGS=-mod=mod",
		)

		if err := compileCmd.Run(); err != nil {
			return fmt.Errorf("compile %s failed: %w", target.label(), err)
		}

		fmt.Printf("Built %s\n", outputPath)
	}
	return nil
}

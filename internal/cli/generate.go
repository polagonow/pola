package cli

import (
	"cmp"
	"fmt"
	"os"

	"github.com/polagonow/pola/internal/autoload"
	"github.com/polagonow/pola/internal/generators"
	_ "github.com/polagonow/pola/internal/generators/all" // register all generators
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

var generateFlags struct {
	actionsDir string
	tsOut      string
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate bridge code from actions/ directory",
	Long: `Parse Go structs in the actions/ directory and generate:
  - Go RuntimeInjector wiring (injected via -overlay, not written to your project)
  - TypeScript declaration file (typed di imports)`,
	RunE: runGenerate,
	Example: `  pola generate
  pola generate --actions-dir ./actions --ts-out ./ui/packages/di/src/generated.d.ts`,
	Aliases: []string{"gen", "g"},
}

func init() {
	generateCmd.Flags().StringVar(&generateFlags.actionsDir, "actions-dir", "", "path to actions directory (default: ./actions)")
	generateCmd.Flags().StringVar(&generateFlags.tsOut, "ts-out", "", "path to generated .d.ts file")
	generateCmd.PersistentFlags().Bool("force", false, "overwrite files that already exist")
	generateCmd.PersistentFlags().Bool("skip-collision-check", false, "skip collision check entirely")
	generateCmd.PersistentFlags().Bool("skip-tests", false, "skip generating test files")

	// Dynamically register all scaffold subcommands from the generator registry.
	for _, g := range generators.All() {
		generateCmd.AddCommand(g.Command())
	}
}

func runGenerate(_ *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	// Load Polafile defaults (env vars take precedence, then Polafile, then hardcoded).
	var pf polafile.Polafile
	pfPtr, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pfPtr != nil {
		pf = *pfPtr
	}

	genOpts := autoload.PluginOpts{
		PolaPackage:     pf.PolaPackage(),
		Engine:          cmp.Or(os.Getenv("POLA_VM"), nameOnly(pf.Engine), "goja"),
		Bundler:         cmp.Or(os.Getenv("POLA_BUNDLER"), nameOnly(pf.Bundler), "esbuild"),
		Renderer:        cmp.Or(os.Getenv("POLA_RENDERER"), nameOnly(pf.Renderer), "react"),
		Router:          cmp.Or(os.Getenv("POLA_ROUTER"), nameOnly(pf.Router), "nextjs"),
		CSS:             cmp.Or(os.Getenv("POLA_CSS"), nameOnly(pf.CSS), "tailwind"),
		Cache:           cmp.Or(os.Getenv("POLA_CACHE"), pf.CacheAdapter("default"), "memory"),
		CSRF:            autoload.EnvOrBool("POLA_CSRF", pf.CSRFEnabled("default")),
		SecurityHeaders: autoload.EnvOrBool("POLA_SECURITY_HEADERS", pf.SecurityHeadersEnabled("default")),
		Dev:             true,
		ActionsDir:      generateFlags.actionsDir,
		TSOut:           generateFlags.tsOut,
	}

	defaultEnv := "development"
	autoload.PopulateDatabaseOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateStorageOpts(&genOpts, &pf, defaultEnv)
	autoload.ApplyMailerOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateMCPOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateSessionOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateJWTOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateProtectOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateCSRFOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateRateLimitOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateFlashOpts(&genOpts, &pf, defaultEnv)
	autoload.PopulateI18nOpts(&genOpts, &pf, defaultEnv)
	result, err := autoload.Run(projectDir, genOpts, verbose)
	if err != nil {
		return err
	}
	if result != nil && result.TmpDir != "" {
		defer os.RemoveAll(result.TmpDir)
	}
	if result != nil && result.OverlayPath != "" {
		fmt.Printf("Overlay: %s\n", result.OverlayPath)
		fmt.Println("Pass -overlay to go build/run to include the generated bridge.")
	}
	return nil
}

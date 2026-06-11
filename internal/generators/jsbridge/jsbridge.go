// Package jsbridge implements the js:bridge generator for the Pola CLI.
// It generates TypeScript bridge declarations from the actions/ directory.
package jsbridge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/polagonow/pola/internal/actionbridge"
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

// JSBridgeGenerator generates TypeScript bridge code from Go actions.
type JSBridgeGenerator struct{}

func init() {
	generators.Register(&JSBridgeGenerator{})
}

func (g *JSBridgeGenerator) Name() string        { return "js:bridge" }
func (g *JSBridgeGenerator) Description() string { return "Generate TypeScript bridge from actions/" }
func (g *JSBridgeGenerator) AfterHooks() []generators.Hook { return nil }

func (g *JSBridgeGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "js:bridge",
		Short: "Generate TypeScript bridge from actions/",
		Long:  "Parse Go action structs and generate TypeScript declarations for client-side usage.",
		RunE:  g.run,
		Example: `  pola generate js:bridge`,
	}
}

func (g *JSBridgeGenerator) run(_ *cobra.Command, _ []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		return fmt.Errorf("no Polafile found in %s", projectDir)
	}
	if pf.IsAPIOnly() {
		fmt.Println("API-only mode: skipping JS bridge generation.")
		return nil
	}

	actionsDir := filepath.Join(projectDir, "actions")
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		fmt.Println("No actions/ directory found, nothing to generate.")
		return nil
	}

	tsOut := filepath.Join(projectDir, pf.AppDir(), "node_modules", "@pola", "actions", "src", "generated.ts")

	tmpDir, err := os.MkdirTemp("", "pola-bridge-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("Generating JS bridge...")
	result, err := actionbridge.Run(actionsDir, tsOut, tmpDir, pf.PolaPackage())
	if err != nil {
		return fmt.Errorf("actionbridge: %w", err)
	}

	if result != nil && result.TSOutPath != "" {
		fmt.Printf("Generated types: %s\n", result.TSOutPath)
	}

	return nil
}

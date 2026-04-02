// Package scaffold implements the scaffold generator for the Pola CLI.
// It composes the model, action, and route generators to create a full
// resource in one command, similar to Rails' scaffold generator.
package scaffold

import (
	"fmt"
	"strings"

	"github.com/polagonow/pola/internal/generators"
	"github.com/spf13/cobra"
)

// ScaffoldGenerator generates a model, action, and route for a resource.
type ScaffoldGenerator struct{}

func init() {
	generators.Register(&ScaffoldGenerator{})
}

func (g *ScaffoldGenerator) Name() string { return "scaffold" }
func (g *ScaffoldGenerator) Description() string {
	return "Generate model, action, and route for a resource"
}
func (g *ScaffoldGenerator) AfterHooks() []generators.Hook { return nil }

func (g *ScaffoldGenerator) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scaffold [Name] [field:type ...]",
		Short: "Generate model, action, and route for a resource",
		Long: `Generate a complete resource scaffold including model, action, and route.
Pass field definitions just like the model generator.

Use --skip-model, --skip-action, or --skip-route to omit specific parts.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Aliases: []string{"s"},
		Example: `  pola generate scaffold Product name:string price:float description:text
  pola generate scaffold Product name:string --skip-route
  pola generate scaffold Product name:string --skip-model --skip-action`,
	}
	cmd.Flags().Bool("skip-model", false, "skip model generation")
	cmd.Flags().Bool("skip-action", false, "skip action generation")
	cmd.Flags().Bool("skip-route", false, "skip route generation")
	return cmd
}

func (g *ScaffoldGenerator) run(cmd *cobra.Command, args []string) error {
	name := args[0]

	skipModel, _ := cmd.Flags().GetBool("skip-model")
	skipAction, _ := cmd.Flags().GetBool("skip-action")
	skipRoute, _ := cmd.Flags().GetBool("skip-route")

	if !skipModel {
		fmt.Printf("Generating model %s...\n", name)
		if err := generators.Run("model", cmd, args); err != nil {
			return fmt.Errorf("model: %w", err)
		}
	}

	if !skipAction {
		fmt.Printf("Generating action %s...\n", name)
		if err := generators.Run("action", cmd, []string{name}); err != nil {
			return fmt.Errorf("action: %w", err)
		}
	}

	if !skipRoute {
		fmt.Printf("Generating route %s...\n", name)
		if err := generators.Run("route", cmd, []string{strings.ToLower(name), "GET,POST,PUT,PATCH,DELETE"}); err != nil {
			return fmt.Errorf("route: %w", err)
		}
	}

	return nil
}

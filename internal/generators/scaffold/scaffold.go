// Package scaffold implements the scaffold generator for the Pola CLI.
// It composes the model, action, and route generators to create a full
// resource in one command, similar to Rails' scaffold generator.
package scaffold

import (
	"fmt"
	"strings"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model"
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
  pola generate scaffold Product name:string --skip-repository --skip-service`,
	}
	cmd.Flags().Bool("skip-model", false, "skip model generation")
	cmd.Flags().Bool("skip-repository", false, "skip repository generation")
	cmd.Flags().Bool("skip-service", false, "skip service generation")
	cmd.Flags().Bool("skip-action", false, "skip action generation")
	cmd.Flags().Bool("skip-route", false, "skip route generation")
	cmd.Flags().Bool("skip-migration", false, "skip migration generation (propagated to model generator)")
	return cmd
}

func (g *ScaffoldGenerator) run(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Detect ID type from field args for downstream generators.
	idGoType := "uint"
	def, err := model.ParseArgs(args)
	if err == nil && def.HasUUIDPrimaryKey() {
		idGoType = "string"
	}
	skipModel, _ := cmd.Flags().GetBool("skip-model")
	skipRepository, _ := cmd.Flags().GetBool("skip-repository")
	skipService, _ := cmd.Flags().GetBool("skip-service")
	skipAction, _ := cmd.Flags().GetBool("skip-action")
	skipRoute, _ := cmd.Flags().GetBool("skip-route")

	if !skipModel {
		fmt.Printf("Generating model %s...\n", name)
		if err := generators.Run("model", cmd, args); err != nil {
			return fmt.Errorf("model: %w", err)
		}
	}

	if !skipRepository {
		fmt.Printf("Generating repository %s...\n", name)
		if err := generators.Run("repository", cmd, args); err != nil {
			return fmt.Errorf("repository: %w", err)
		}
	}

	if !skipService {
		fmt.Printf("Generating service %s...\n", name)
		if err := generators.Run("service", cmd, args); err != nil {
			return fmt.Errorf("service: %w", err)
		}
	}

	if !skipAction {
		fmt.Printf("Generating action %s...\n", name)
		actionArgs := []string{name}
		if !skipService {
			actionArgs = append(actionArgs, "--service="+name)
		}
		actionArgs = append(actionArgs, "--id-type="+idGoType)
		if err := generators.Run("action", cmd, actionArgs); err != nil {
			return fmt.Errorf("action: %w", err)
		}
	}

	if !skipRoute {
		fmt.Printf("Generating route %s...\n", name)
		routeArgs := []string{strings.ToLower(name), "GET,POST,PUT,PATCH,DELETE"}
		if !skipService {
			routeArgs = append(routeArgs, "--service="+name)
		}
		routeArgs = append(routeArgs, "--id-type="+idGoType)
		if err := generators.Run("route", cmd, routeArgs); err != nil {
			return fmt.Errorf("route: %w", err)
		}
	}

	return nil
}

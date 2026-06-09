// Package scaffold implements the scaffold generator for the Pola CLI.
// It composes the model, action, and route generators to create a full
// resource in one command, similar to Rails' scaffold generator.
package scaffold

import (
	"fmt"
	"strings"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model"
	"github.com/polagonow/pola/internal/generators/model/schema"
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
	cmd.Flags().Bool("skip-zod", false, "skip Zod schema generation")
	cmd.Flags().Bool("skip-views", false, "skip page generation")
	cmd.Flags().Bool("skip-migration", false, "skip migration generation (propagated to model generator)")
	return cmd
}

func (g *ScaffoldGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("scaffold name is required")
	}
	name := args[0]

	skipModel, _ := cmd.Flags().GetBool("skip-model")
	skipRepository, _ := cmd.Flags().GetBool("skip-repository")
	skipService, _ := cmd.Flags().GetBool("skip-service")
	skipAction, _ := cmd.Flags().GetBool("skip-action")
	skipRoute, _ := cmd.Flags().GetBool("skip-route")
	skipZod, _ := cmd.Flags().GetBool("skip-zod")
	skipViews, _ := cmd.Flags().GetBool("skip-views")

	var all []string

	collect := func(genName string, genArgs []string) error {
		g, err := generators.Get(genName)
		if err != nil {
			return err
		}
		d, ok := g.(generators.Destroyer)
		if !ok {
			return nil
		}
		paths, err := d.Artifacts(cmd, genArgs, projectDir)
		if err != nil {
			return err
		}
		all = append(all, paths...)
		return nil
	}

	if !skipModel {
		if err := collect("model", args); err != nil {
			return nil, fmt.Errorf("model artifacts: %w", err)
		}
	}
	if !skipRepository {
		if err := collect("repository", args); err != nil {
			return nil, fmt.Errorf("repository artifacts: %w", err)
		}
	}
	if !skipService {
		if err := collect("service", args); err != nil {
			return nil, fmt.Errorf("service artifacts: %w", err)
		}
	}
	if !skipAction {
		actionArgs := []string{name}
		if err := collect("action", actionArgs); err != nil {
			return nil, fmt.Errorf("action artifacts: %w", err)
		}
	}
	if !skipRoute {
		routeArgs := []string{strings.ToLower(name), "GET,POST,PUT,PATCH,DELETE"}
		if err := collect("route", routeArgs); err != nil {
			return nil, fmt.Errorf("route artifacts: %w", err)
		}
	}
	if !skipZod {
		if err := collect("zod", args); err != nil {
			return nil, fmt.Errorf("zod artifacts: %w", err)
		}
	}
	if !skipViews {
		if err := collect("page", args); err != nil {
			return nil, fmt.Errorf("page artifacts: %w", err)
		}
	}

	return all, nil
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
		if def.HasBlobFields() {
			routeArgs = append(routeArgs, "--has-file-upload")
			var fileFields []string
			for _, f := range def.BlobFields() {
				fileFields = append(fileFields, schema.SnakeCase(f.Name))
			}
			routeArgs = append(routeArgs, "--file-fields="+strings.Join(fileFields, ","))
		}
		if err := generators.Run("route", cmd, routeArgs); err != nil {
			return fmt.Errorf("route: %w", err)
		}
	}

	skipZod, _ := cmd.Flags().GetBool("skip-zod")
	if !skipZod {
		fmt.Printf("Generating zod schema %s...\n", name)
		if err := generators.Run("zod", cmd, args); err != nil {
			return fmt.Errorf("zod: %w", err)
		}
	}

	skipViews, _ := cmd.Flags().GetBool("skip-views")
	if !skipViews {
		fmt.Printf("Generating pages %s...\n", name)
		if err := generators.Run("page", cmd, args); err != nil {
			return fmt.Errorf("page: %w", err)
		}
	}

	return nil
}

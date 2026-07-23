// Package seed implements the database-seed scaffold generator for the Pola CLI.
// It creates db/seeds/seeds.go, which `pola db seed` runs.
package seed

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/project"
	"github.com/spf13/cobra"
)

const seedFileContent = `// Package seeds populates the database with initial data.
//
// Run it with:  pola db seed
package seeds

import (
	"context"

	"github.com/polagonow/pola/core"
)

// Seed runs with the full DI registry, so it can resolve services and
// repositories via core.MustInvoke and create records. Guard inserts so that
// re-running the seeder does not duplicate data.
//
// Example:
//
//	users := core.MustInvoke[*services.UserService](r)
//	_ = users.Create(ctx, &models.User{Email: "test@test.com"})
func Seed(ctx context.Context, r *core.Registry) error {
	_ = ctx
	_ = r
	return nil
}
`

// SeedGenerator scaffolds db/seeds/seeds.go.
type SeedGenerator struct{}

func (g *SeedGenerator) Name() string        { return "seed" }
func (g *SeedGenerator) Description() string { return "Scaffold a database seeder (db/seeds)" }

func (g *SeedGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{generators.CmdHook("gofmt", "-w", ".")}
}

func (g *SeedGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Scaffold a database seeder (db/seeds)",
		Long: `Create db/seeds/seeds.go with a Seed(ctx, *core.Registry) function.

The framework wires it automatically; run the seeder with 'pola db seed'.`,
		Args:    cobra.NoArgs,
		RunE:    g.run,
		Example: `  pola generate seed`,
	}
}

func (g *SeedGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	return []string{filepath.Join(projectDir, "db", "seeds", "seeds.go")}, nil
}

func (g *SeedGenerator) run(cmd *cobra.Command, args []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}
	seedsDir := filepath.Join(projectDir, "db", "seeds")
	if err := os.MkdirAll(seedsDir, 0o755); err != nil {
		return fmt.Errorf("create seeds dir: %w", err)
	}
	filePath := filepath.Join(seedsDir, "seeds.go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, []byte(seedFileContent), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Printf("Created %s\n", filePath)
	fmt.Println("Run it with: pola db seed")
	return generators.RunAfterHooks(g, projectDir)
}

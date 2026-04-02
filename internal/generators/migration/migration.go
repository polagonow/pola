// Package migration implements the migration generator for the Pola CLI.
// It diffs ORM model schemas against the migration directory using Atlas
// to auto-generate versioned SQL migration files.
package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/migration/diff"
	entdiff "github.com/polagonow/pola/internal/generators/migration/ent"
	gormdiff "github.com/polagonow/pola/internal/generators/migration/gorm"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

// MigrationGenerator generates versioned migration files by diffing ORM models.
type MigrationGenerator struct{}

func init() {
	// Register ORM diff generators.
	diff.Register(&entdiff.EntDiffGenerator{})
	diff.Register(&gormdiff.GormDiffGenerator{})

	// Register this generator with the CLI generator registry.
	generators.Register(&MigrationGenerator{})
}

func (g *MigrationGenerator) Name() string                  { return "migration" }
func (g *MigrationGenerator) Description() string           { return "Generate a versioned migration by diffing models" }
func (g *MigrationGenerator) AfterHooks() []generators.Hook { return nil }

func (g *MigrationGenerator) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migration [name]",
		Short:   "Generate a versioned migration by diffing models",
		Long:    `Diff ORM model schemas against the migration directory and auto-generate a versioned SQL migration file using Atlas.`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"mi"},
		RunE:    g.run,
		Example: `  pola generate migration CreateUsers
  pola generate migration AddEmailToUsers --env development
  pola generate migration init --dev-url "sqlite://file?mode=memory"`,
	}

	cmd.Flags().String("env", "development", "Environment to use for adapter/dev-url resolution")
	cmd.Flags().String("dev-url", "", "Dev database URL (overrides Polafile and auto-detect)")

	return cmd
}

func (g *MigrationGenerator) run(cmd *cobra.Command, args []string) error {
	name := toSnakeCase(args[0])

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	// Load Polafile.
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		return fmt.Errorf("no Polafile.hcl found; run 'pola new' first or create one manually")
	}
	if pf.Database == nil {
		return fmt.Errorf("no database block in Polafile.hcl; run 'pola generate model' first to configure the database")
	}

	env, _ := cmd.Flags().GetString("env")
	devURLFlag, _ := cmd.Flags().GetString("dev-url")

	orm := pf.DatabaseORM()
	adapter := pf.DatabaseAdapter(env)
	modelsDir := pf.DatabaseModelsDir()
	migrationsDir := filepath.Join(projectDir, pf.DatabaseMigrationsDir())

	// Read module path from go.mod.
	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	// Resolve dev URL.
	devURL, err := resolveDevURL(devURLFlag, pf, env, adapter)
	if err != nil {
		return err
	}

	// Ensure migrations directory exists.
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return fmt.Errorf("create migrations dir: %w", err)
	}

	cfg := diff.Config{
		Name:          name,
		ProjectDir:    projectDir,
		ModulePath:    modulePath,
		GoVersion:     project.GoVersion(projectDir),
		ModelsDir:     modelsDir,
		MigrationsDir: migrationsDir,
		DevURL:        devURL,
		Adapter:       adapter,
	}

	ctx := context.Background()

	// Lookup and run the ORM-specific diff generator.
	gen, err := diff.Get(orm)
	if err != nil {
		return err
	}
	if err := gen.Diff(ctx, cfg); err != nil {
		return err
	}

	// Generate schema.hcl (non-fatal if it fails).
	if err := generateSchemaHCL(ctx, cfg); err != nil {
		fmt.Printf("Warning: could not generate schema.hcl: %v\n", err)
	}

	return nil
}

// toSnakeCase converts PascalCase/camelCase to snake_case.
// "CreateProducts" → "create_products"
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

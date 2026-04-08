// Package model implements the model scaffold generator for the Pola CLI.
// It contains all model generation logic including field parsing, validation,
// type mapping, and ORM plugin registration (ent, gorm).
package model

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model/beego"
	"github.com/polagonow/pola/internal/generators/model/ent"
	"github.com/polagonow/pola/internal/generators/model/gorm"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

// ModelGenerator generates ORM model/schema files using the configured ORM plugin.
type ModelGenerator struct{}

func init() {
	// Register ORM generators.
	schema.RegisterORMGenerator(&beego.BeegoGenerator{})
	schema.RegisterORMGenerator(&ent.EntGenerator{})
	schema.RegisterORMGenerator(&gorm.GormGenerator{})

	// Register this generator with the CLI generator registry.
	generators.Register(&ModelGenerator{})
}

func (g *ModelGenerator) Name() string { return "model" }
func (g *ModelGenerator) Description() string {
	return "Generate an ORM model/schema from field definitions"
}

func (g *ModelGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("go", "mod", "tidy"),
		generators.CmdHook("gofmt", "-w", "."),
	}
}

func (g *ModelGenerator) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model [Name] [field:type ...]",
		Short: "Generate an ORM model/schema from field definitions",
		Long: `Parse model name and field:type{opts}:modifier specs, then generate
ORM-specific schema files using the plugin configured in Polafile.hcl.

Field syntax: field:type{options}:modifier1:modifier2
  Types:    string, int, int64, float, bool, time, uuid, text, bytes, json, references
  Options:  {polymorphic} (only on references)
  Modifiers: index, uniq`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Aliases: []string{"m"},
		Example: `  pola generate model User name:string email:string:uniq age:int
  pola generate model Article title:string:index body:text author:references
  pola generate model Comment body:text commentable:references{polymorphic}`,
	}
	cmd.Flags().Bool("skip-migration", false, "Skip auto-generating a migration after model creation")
	return cmd
}

func (g *ModelGenerator) run(cmd *cobra.Command, args []string) error {
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
		pf = &polafile.Polafile{}
	}

	// Ensure database block exists; prompt interactively for missing ORM/adapter.
	dirty := false
	if pf.Database == nil {
		pf.Database = &polafile.Database{
			Models: "db/models",
			Migrations: &polafile.Migrations{
				Directory: "db/migrations",
				Format:    "hcl",
			},
			Envs: []polafile.DatabaseEnvironment{
				{Environment: "development", Adapter: "sqlite"},
				{Environment: "production", Adapter: "postgresql"},
			},
		}
		dirty = true
	}
	if pf.Database.ORM == "" {
		orm, err := promptSelect("ORM:", []string{"beego", "ent", "gorm"})
		if err != nil {
			return err
		}
		pf.Database.ORM = orm
		dirty = true
	}
	if pf.Database.Adapter == "" {
		adapter, err := promptSelect("Database adapter:", []string{"postgresql", "mysql", "sqlite"})
		if err != nil {
			return err
		}
		pf.Database.Adapter = adapter
		dirty = true
	}
	if dirty {
		if err := polafile.Save(projectDir, pf); err != nil {
			return fmt.Errorf("save Polafile: %w", err)
		}
		fmt.Println("Saved database configuration to Polafile.hcl.")
	}

	ModelDefinition, err := ParseArgs(args)
	if err != nil {
		return err
	}

	orm := pf.DatabaseORM()
	outDir := filepath.Join(projectDir, pf.DatabaseModelsDir())

	// Validate that referenced models exist and resolve FK types.
	if err := ValidateReferences(ModelDefinition, outDir, orm); err != nil {
		return err
	}

	gen, err := schema.GetORMGenerator(orm)
	if err != nil {
		return err
	}

	// Ent schemas are placed under "schema/" subdirectory, other ORMs use their name.
	subDir := orm
	if orm == "ent" {
		subDir = "schema"
	}
	outFile := filepath.Join(outDir, subDir, schema.SnakeCase(ModelDefinition.Name)+".go")

	if err := generators.CheckCollision(cmd, outFile); err != nil {
		return err
	}

	if err := gen.Generate(ModelDefinition, outDir); err != nil {
		return fmt.Errorf("generate model: %w", err)
	}

	fmt.Printf("Created %s\n", outFile)

	if orm == "ent" {
		fmt.Println("Running ent codegen...")
		if err := runEntCodegen(projectDir, pf.DatabaseModelsDir(), pf.DatabaseEntClientDir()); err != nil {
			return fmt.Errorf("ent codegen: %w", err)
		}
	}

	if err := generators.RunAfterHooks(g, projectDir); err != nil {
		return err
	}

	// Auto-generate migration unless --skip-migration is set.
	skipMigration, _ := cmd.Flags().GetBool("skip-migration")
	if !skipMigration {
		migName := "Create" + schema.Pluralize(ModelDefinition.Name)
		fmt.Printf("Generating migration %s...\n", migName)
		if err := generators.Run("migration", cmd, []string{migName}); err != nil {
			return fmt.Errorf("migration: %w", err)
		}
	}

	return nil
}

// runEntCodegen runs the ent code generator to produce the typed client package.
func runEntCodegen(projectDir, modelsDir, entClientDir string) error {
	targetDir := filepath.Join(projectDir, entClientDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create ent client dir: %w", err)
	}
	// Seed a package file so ent can resolve the Go package name for the target directory.
	seedFile := filepath.Join(targetDir, "doc.go")
	if _, err := os.Stat(seedFile); os.IsNotExist(err) {
		if err := os.WriteFile(seedFile, []byte("package ent\n"), 0o644); err != nil {
			return fmt.Errorf("write ent seed file: %w", err)
		}
	}
	schemaPath := "./" + modelsDir + "/schema"
	cmd := exec.Command("go", "run", "-mod=mod", "entgo.io/ent/cmd/ent", "generate",
		"--target", targetDir,
		schemaPath,
	)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go run entgo.io/ent/cmd/ent generate: %w", err)
	}
	return nil
}

// promptSelect presents an interactive selection prompt and returns the chosen option.
func promptSelect(message string, options []string) (string, error) {
	var answer string
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}
	return answer, nil
}

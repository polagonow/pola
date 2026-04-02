// Package beego implements the Beego ORM migration diff generator.
// It creates a temporary Go program that imports the user's Beego models
// and uses the Atlas provider to auto-generate versioned SQL migrations.
package beego

import (
	"context"
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/generators/migration/diff"
)

//go:embed all:_templates
var templates embed.FS

var diffTmpl = template.Must(
	template.New("beego_diff_main.go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/beego_diff_main.go.tmpl"),
)

// BeegoDiffGenerator generates migrations by diffing Beego model structs.
type BeegoDiffGenerator struct{}

func (g *BeegoDiffGenerator) Name() string { return "beego" }

func (g *BeegoDiffGenerator) Diff(ctx context.Context, cfg diff.Config) error {
	// Discover Beego model struct names from the models directory.
	beegoDir := filepath.Join(cfg.ProjectDir, cfg.ModelsDir, "beego")
	models, err := discoverBeegoModels(beegoDir)
	if err != nil {
		return fmt.Errorf("discover beego models: %w", err)
	}
	if len(models) == 0 {
		return fmt.Errorf("no Beego models found in %s", beegoDir)
	}

	tmpDir, err := os.MkdirTemp("", "pola-beego-diff-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate main.go from template.
	data := struct {
		ModulePath   string
		ModelsDir    string
		DriverImport string
		AtlasDialect string
		BeegoDialect string
		Models       []string
	}{
		ModulePath:   cfg.ModulePath,
		ModelsDir:    cfg.ModelsDir,
		DriverImport: diff.DriverImportForAdapter(cfg.Adapter),
		AtlasDialect: diff.AtlasDialectForAdapter(cfg.Adapter),
		BeegoDialect: beegoDialectForAdapter(cfg.Adapter),
		Models:       models,
	}

	var buf strings.Builder
	if err := diffTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute beego diff template: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// Write go.mod with replace directive pointing to the user's project.
	goMod := fmt.Sprintf("module pola-beego-diff\n\ngo %s\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
		cfg.GoVersion, cfg.ModulePath, cfg.ModulePath, cfg.ProjectDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	// Run go mod tidy to resolve dependencies.
	tidy := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidy.Dir = tmpDir
	if out, err := tidy.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy: %s\n%w", out, err)
	}

	// Run the temporary program: args: migrationsDir, migrationName
	run := exec.CommandContext(ctx, "go", "run", ".", cfg.MigrationsDir, cfg.Name)
	run.Dir = tmpDir
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Run(); err != nil {
		return fmt.Errorf("beego migration diff: %w", err)
	}

	return nil
}

// discoverBeegoModels scans Go files in dir for exported struct types
// (Beego models) and returns their names.
func discoverBeegoModels(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var models []string
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); ok {
					models = append(models, ts.Name.Name)
				}
			}
		}
	}
	return models, nil
}

// beegoDialectForAdapter returns the Beego ORM driver name for the given adapter.
// Beego uses "sqlite3", "postgres", "mysql" as driver names.
func beegoDialectForAdapter(adapter string) string {
	switch adapter {
	case "sqlite":
		return "sqlite3"
	case "postgresql", "postgres":
		return "postgres"
	case "mysql":
		return "mysql"
	default:
		return "postgres"
	}
}

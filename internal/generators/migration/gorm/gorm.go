// Package gorm implements the GORM ORM migration diff generator.
// It creates a temporary Go program that imports the user's GORM models
// and uses the Atlas provider to auto-generate versioned SQL migrations.
package gorm

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
	template.New("gorm_diff_main.go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/gorm_diff_main.go.tmpl"),
)

// GormDiffGenerator generates migrations by diffing GORM model structs.
type GormDiffGenerator struct{}

func (g *GormDiffGenerator) Name() string { return "gorm" }

func (g *GormDiffGenerator) Diff(ctx context.Context, cfg diff.Config) error {
	// Discover GORM model struct names from the models directory.
	gormDir := filepath.Join(cfg.ProjectDir, cfg.ModelsDir, "gorm")
	models, err := discoverGormModels(gormDir)
	if err != nil {
		return fmt.Errorf("discover gorm models: %w", err)
	}
	if len(models) == 0 {
		return fmt.Errorf("no GORM models found in %s", gormDir)
	}

	tmpDir, err := os.MkdirTemp("", "pola-gorm-diff-*")
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
		Models       []string
	}{
		ModulePath:   cfg.ModulePath,
		ModelsDir:    cfg.ModelsDir,
		DriverImport: diff.DriverImportForAdapter(cfg.Adapter),
		AtlasDialect: diff.AtlasDialectForAdapter(cfg.Adapter),
		Models:       models,
	}

	var buf strings.Builder
	if err := diffTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute gorm diff template: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// Write go.mod with replace directive pointing to the user's project.
	if err := diff.WriteGoMod(tmpDir, "pola-gorm-diff", cfg); err != nil {
		return err
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
		return fmt.Errorf("gorm migration diff: %w", err)
	}

	return nil
}

// discoverGormModels scans Go files in dir for exported struct types
// (GORM models) and returns their names.
func discoverGormModels(dir string) ([]string, error) {
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

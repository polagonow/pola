// Package gorm implements the GORM ORM migration diff generator.
// It creates a temporary Go program that imports the user's GORM models
// and uses the Atlas provider to auto-generate versioned SQL migrations.
package gorm

import (
	"context"
	"embed"
	"fmt"
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
	}{
		ModulePath:   cfg.ModulePath,
		ModelsDir:    cfg.ModelsDir,
		DriverImport: diff.DriverImportForAdapter(cfg.Adapter),
		AtlasDialect: diff.AtlasDialectForAdapter(cfg.Adapter),
	}

	var buf strings.Builder
	if err := diffTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute gorm diff template: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// Write go.mod with replace directive pointing to the user's project.
	// Dependencies (atlas, atlas-provider-gorm) are resolved transitively from the user's module.
	goMod := fmt.Sprintf("module pola-gorm-diff\n\ngo %s\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
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

	// Run the temporary program.
	run := exec.CommandContext(ctx, "go", "run", ".", cfg.MigrationsDir, cfg.DevURL, cfg.Name)
	run.Dir = tmpDir
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Run(); err != nil {
		return fmt.Errorf("gorm migration diff: %w", err)
	}

	return nil
}

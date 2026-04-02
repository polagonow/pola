// Package ent implements the Ent ORM migration diff generator.
// It creates a temporary Go program that uses entc.LoadGraph to load
// the user's ent schemas and schema.Diff to auto-generate versioned
// SQL migrations — no ent codegen required.
package ent

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
	template.New("ent_diff_main.go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/ent_diff_main.go.tmpl"),
)

// EntDiffGenerator generates migrations by diffing Ent schema structs.
type EntDiffGenerator struct{}

func (g *EntDiffGenerator) Name() string { return "ent" }

func (g *EntDiffGenerator) Diff(ctx context.Context, cfg diff.Config) error {
	tmpDir, err := os.MkdirTemp("", "pola-ent-diff-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate main.go from template.
	data := struct {
		ModulePath   string
		ModelsDir    string
		DriverImport string
		Dialect      string
	}{
		ModulePath:   cfg.ModulePath,
		ModelsDir:    cfg.ModelsDir,
		DriverImport: diff.DriverImportForAdapter(cfg.Adapter),
		Dialect:      diff.DialectForAdapter(cfg.Adapter),
	}

	var buf strings.Builder
	if err := diffTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute ent diff template: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// Write go.mod with replace directive pointing to the user's project.
	// Dependencies (atlas, ent) are resolved transitively from the user's module.
	goMod := fmt.Sprintf("module pola-ent-diff\n\ngo %s\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
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

	// Schema import path for entc.LoadGraph.
	schemaImportPath := cfg.ModulePath + "/" + cfg.ModelsDir + "/ent"

	// Run the temporary program:
	// args: migrationsDir, devURL, migrationName, schemaImportPath
	run := exec.CommandContext(ctx, "go", "run", ".", cfg.MigrationsDir, cfg.DevURL, cfg.Name, schemaImportPath)
	run.Dir = tmpDir
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Run(); err != nil {
		return fmt.Errorf("ent migration diff: %w", err)
	}

	return nil
}

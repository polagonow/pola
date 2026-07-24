// Package dbseed implements the database-seed overlay autoload. When a project
// has a db/seeds package exporting `func Seed(ctx, *core.Registry) error`, it
// generates a pola_seed.go overlay whose init() registers that function with the
// framework's seed registry, so `pola db seed` (POLA_SEED_ONLY=true) can run it
// after the app is built.
package dbseed

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
)

//go:embed _templates/pola_seed.go.tmpl
var templates embed.FS

var seedTmpl = template.Must(
	template.New("pola_seed.go.tmpl").ParseFS(templates, "_templates/pola_seed.go.tmpl"),
)

type autoloadImpl struct{}

// New returns this autoload stage for explicit registration in autoload/all.
func New() autoload.Autoload { return &autoloadImpl{} }

func (a *autoloadImpl) Name() string { return "dbseed" }

// Priority is independent of pluginimports (the generated overlay self-registers
// via init()), so any slot works; keep it near dbembed.
func (a *autoloadImpl) Priority() int { return 860 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if ctx.ModPath == "" {
		return nil
	}
	seedsDir := filepath.Join(ctx.ProjectDir, "db", "seeds")
	if !hasSeedFunc(seedsDir) {
		return nil
	}

	data := struct {
		PolaPackage string
		SeedsImport string
	}{
		PolaPackage: ctx.Opts.PolaPackage,
		SeedsImport: ctx.ModPath + "/db/seeds",
	}

	var buf strings.Builder
	if err := seedTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute seed template: %w", err)
	}

	absProjectDir, err := filepath.Abs(ctx.ProjectDir)
	if err != nil {
		return fmt.Errorf("abs project dir: %w", err)
	}
	seedPath := filepath.Join(ctx.TmpDir, "pola_seed.go")
	if err := os.WriteFile(seedPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write pola_seed.go: %w", err)
	}
	ctx.Replace[filepath.Join(absProjectDir, "pola_seed.go")] = seedPath
	return nil
}

// hasSeedFunc reports whether dir contains a non-test Go file declaring a
// package-level `func Seed(...)`.
func hasSeedFunc(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "Seed" {
				return true
			}
		}
	}
	return false
}

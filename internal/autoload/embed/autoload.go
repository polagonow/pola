// Package embed implements the asset embedding overlay autoload for production builds.
package embed

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
)

//go:embed _templates/embed_go.tmpl
var templates embed.FS

var embedTmpl = template.Must(
	template.New("embed_go.tmpl").ParseFS(templates, "_templates/embed_go.tmpl"),
)

type autoloadImpl struct{}

// New returns this autoload stage for explicit registration in autoload/all.
func New() autoload.Autoload { return &autoloadImpl{} }

func (a *autoloadImpl) Name() string  { return "embed" }
func (a *autoloadImpl) Priority() int { return 950 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if !ctx.Opts.Embed || ctx.Opts.APIOnly {
		return nil
	}

	var buf strings.Builder
	if err := embedTmpl.Execute(&buf, struct{ PolaPackage string }{ctx.Opts.PolaPackage}); err != nil {
		return fmt.Errorf("execute embed template: %w", err)
	}

	embedPath := filepath.Join(ctx.TmpDir, "pola_embed.go")
	if err := os.WriteFile(embedPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write embed: %w", err)
	}

	absProjectDir, err := filepath.Abs(ctx.ProjectDir)
	if err != nil {
		return fmt.Errorf("abs project dir: %w", err)
	}
	ctx.Replace[filepath.Join(absProjectDir, "pola_embed.go")] = embedPath

	return nil
}

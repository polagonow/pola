// Package entclient generates a Pola plugin that provides *ent.Client in the
// DI container, constructed from the framework-provided *entsql.Driver.
//
// This is separated from the repos autoload because it's a DI-provisioning
// concern (making *ent.Client resolvable via core.MustInvoke), not a
// repositories concern. It runs after the repos autoload so it can piggy-back
// on the ent client directory the polafile already exposes, and it emits the
// plugin into the ent repositories package (repositories/ent/) as a distinct
// pola_ent_client_plugin.go file — keeping the concerns separate in source
// even though they share a Go package.
package entclient

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
)

//go:embed _templates/ent_client_plugin_go.tmpl
var templates embed.FS

var entClientTmpl = template.Must(
	template.New("ent_client_plugin_go.tmpl").ParseFS(templates, "_templates/ent_client_plugin_go.tmpl"),
)

type autoloadImpl struct{}

// New returns this autoload stage for explicit registration in autoload/all.
func New() autoload.Autoload { return &autoloadImpl{} }

func (a *autoloadImpl) Name() string  { return "entclient" }
func (a *autoloadImpl) Priority() int { return 350 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	rd := ctx.Discovery.RepoDisco
	if rd == nil || rd.ORM != "ent" || len(rd.Repositories) == 0 {
		return nil
	}
	if rd.ModulePath == "" || rd.EntClientDir == "" {
		return nil
	}

	entClientImport := rd.ModulePath + "/" + filepath.ToSlash(rd.EntClientDir)

	var buf strings.Builder
	if err := entClientTmpl.Execute(&buf, struct {
		PolaPackage     string
		PkgName         string
		EntClientImport string
	}{
		PolaPackage:     ctx.Opts.PolaPackage,
		PkgName:         rd.PkgName,
		EntClientImport: entClientImport,
	}); err != nil {
		return fmt.Errorf("execute ent client plugin template: %w", err)
	}

	outPath := filepath.Join(ctx.TmpDir, "pola_ent_client_plugin.go")
	if err := os.WriteFile(outPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write ent client plugin: %w", err)
	}

	entAbsDir, _ := filepath.Abs(filepath.Join(ctx.ProjectDir, rd.RepoDir, rd.ORM))
	ctx.Replace[filepath.Join(entAbsDir, "pola_ent_client_plugin.go")] = outPath

	ctx.Discovery.EntClientDisco = &autoload.EntClientDiscovery{
		PkgName:    rd.PkgName,
		ImportPath: rd.ImportPath,
	}
	return nil
}

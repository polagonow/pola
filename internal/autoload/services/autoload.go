// Package services implements the service plugin discovery and overlay autoload.
package services

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/polafile"
)

//go:embed _templates/svc_plugins_go.tmpl
var templates embed.FS

var svcPluginsTmpl = template.Must(
	template.New("svc_plugins_go.tmpl").ParseFS(templates, "_templates/svc_plugins_go.tmpl"),
)

type autoloadImpl struct{}

func init() {
	autoload.Register(&autoloadImpl{})
}

func (a *autoloadImpl) Name() string { return "services" }
func (a *autoloadImpl) Priority() int { return 400 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if ctx.ModPath == "" {
		return nil
	}

	pf, _ := polafile.Load(ctx.ProjectDir)
	svcDir := "services"
	if pf != nil {
		svcDir = pf.ServicesDir()
	}

	svcDisco := discoverServiceConstructors(ctx.ProjectDir, svcDir, ctx.ModPath)
	if svcDisco == nil {
		return nil
	}

	ctx.Discovery.SvcDisco = svcDisco

	var buf strings.Builder
	if err := svcPluginsTmpl.Execute(&buf, struct {
		PolaPackage string
		PkgName     string
		RepoImport  string
		Services    []autoload.PluginEntry
	}{
		PolaPackage: ctx.Opts.PolaPackage,
		PkgName:     svcDisco.PkgName,
		RepoImport:  svcDisco.RepoImport,
		Services:    svcDisco.Services,
	}); err != nil {
		return fmt.Errorf("execute svc plugins template: %w", err)
	}

	svcPluginsPath := filepath.Join(ctx.TmpDir, "pola_svc_plugins.go")
	if err := os.WriteFile(svcPluginsPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write svc plugins: %w", err)
	}

	svcAbsDir, _ := filepath.Abs(filepath.Join(ctx.ProjectDir, svcDisco.PkgName))
	ctx.Replace[filepath.Join(svcAbsDir, "pola_plugins.go")] = svcPluginsPath

	return nil
}

// discoverServiceConstructors scans the services directory for exported
// New*Service constructor functions and returns their names.
func discoverServiceConstructors(projectDir, svcDir, modPath string) *autoload.SvcDiscovery {
	absDir := filepath.Join(projectDir, svcDir)
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil
	}

	var svcs []autoload.PluginEntry
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(absDir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || !fd.Name.IsExported() || fd.Recv != nil {
				continue
			}
			name := fd.Name.Name
			if strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Service") {
				svcName := strings.TrimPrefix(name, "New")
				svcName = strings.TrimSuffix(svcName, "Service")
				if svcName != "" {
					svcs = append(svcs, autoload.PluginEntry{
						Name:      svcName,
						SnakeName: schema.SnakeCase(svcName),
					})
				}
			}
		}
	}

	if len(svcs) == 0 {
		return nil
	}

	pf, _ := polafile.Load(projectDir)
	repoDir := "repositories"
	if pf != nil {
		repoDir = pf.RepositoriesDir()
	}

	sort.Slice(svcs, func(i, j int) bool { return svcs[i].Name < svcs[j].Name })
	return &autoload.SvcDiscovery{
		ImportPath: modPath + "/" + filepath.ToSlash(svcDir),
		RepoImport: modPath + "/" + filepath.ToSlash(repoDir),
		PkgName:    filepath.Base(svcDir),
		Services:   svcs,
	}
}

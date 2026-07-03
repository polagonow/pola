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
	"github.com/polagonow/pola/internal/autoload/ctorscan"
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

func (a *autoloadImpl) Name() string  { return "services" }
func (a *autoloadImpl) Priority() int { return 400 }

// tmplData is the value passed to the service plugins template. It embeds the
// discovery so template range/if references stay stable.
type tmplData struct {
	PolaPackage string
	*autoload.SvcDiscovery
}

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if ctx.ModPath == "" {
		return nil
	}

	pf, _ := polafile.Load(ctx.ProjectDir)
	svcDir := "services"
	if pf != nil {
		svcDir = pf.ServicesDir()
	}

	svcDisco, err := discoverServiceConstructors(ctx.ProjectDir, svcDir, ctx.ModPath, ctx.Opts.PolaPackage)
	if err != nil {
		return err
	}
	if svcDisco == nil {
		return nil
	}

	ctx.Discovery.SvcDisco = svcDisco

	var buf strings.Builder
	if err := svcPluginsTmpl.Execute(&buf, tmplData{PolaPackage: ctx.Opts.PolaPackage, SvcDiscovery: svcDisco}); err != nil {
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

type discoveredSvc struct {
	name     string
	filename string
	fd       *ast.FuncDecl
	file     *ast.File
}

// discoverServiceConstructors scans the services directory for exported
// New*Service constructor functions, extracts their parameters via ctorscan,
// and returns merged discovery data. Returns (nil, nil) when there are no
// services; returns an error when a constructor uses an unsupported param.
func discoverServiceConstructors(projectDir, svcDir, modPath, polaPackage string) (*autoload.SvcDiscovery, error) {
	absDir := filepath.Join(projectDir, svcDir)
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, nil
	}

	var found []discoveredSvc
	interfaceNames := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(absDir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() || d.Recv != nil {
					continue
				}
				name := d.Name.Name
				if strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Service") {
					svcName := strings.TrimPrefix(name, "New")
					svcName = strings.TrimSuffix(svcName, "Service")
					if svcName == "" {
						continue
					}
					found = append(found, discoveredSvc{name: svcName, filename: entry.Name(), fd: d, file: f})
				}
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					if _, ok := ts.Type.(*ast.InterfaceType); ok {
						interfaceNames[ts.Name.Name] = struct{}{}
					}
				}
			}
		}
	}

	if len(found) == 0 {
		return nil, nil
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].name != found[j].name {
			return found[i].name < found[j].name
		}
		return found[i].filename < found[j].filename
	})

	seen := map[string]struct{}{}
	var svcs []autoload.PluginEntry
	var allParams []ctorscan.Param
	for _, d := range found {
		if _, dup := seen[d.name]; dup {
			continue
		}
		seen[d.name] = struct{}{}
		params, err := ctorscan.ScanParams(d.fd, d.file, d.filename, polaPackage)
		if err != nil {
			return nil, fmt.Errorf("autoload services: %w", err)
		}
		allParams = append(allParams, params...)
		_, hasIface := interfaceNames[d.name+"ServiceInterface"]
		svcs = append(svcs, autoload.PluginEntry{
			Name:         d.name,
			SnakeName:    schema.SnakeCase(d.name),
			Params:       params,
			Args:         renderArgs(params),
			HasInterface: hasIface,
		})
	}

	skip := map[string]struct{}{polaPackage + "/core": {}}
	extra, _ := ctorscan.MergeImports(allParams, skip)

	pf, _ := polafile.Load(projectDir)
	repoDir := "repositories"
	if pf != nil {
		repoDir = pf.RepositoriesDir()
	}

	return &autoload.SvcDiscovery{
		ImportPath:   modPath + "/" + filepath.ToSlash(svcDir),
		RepoImport:   modPath + "/" + filepath.ToSlash(repoDir),
		PkgName:      filepath.Base(svcDir),
		Services:     svcs,
		ExtraImports: extra,
	}, nil
}

// renderArgs joins constructor argument names in declaration order: "r" for
// registry params, "p0", "p1", ... for others.
func renderArgs(params []ctorscan.Param) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		if p.IsRegistry {
			parts[i] = "r"
		} else {
			parts[i] = fmt.Sprintf("p%d", i)
		}
	}
	return strings.Join(parts, ", ")
}

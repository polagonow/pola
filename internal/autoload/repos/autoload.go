// Package repos implements the repository plugin discovery and overlay autoload.
package repos

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

//go:embed _templates/repo_plugins_go.tmpl
var templates embed.FS

var repoPluginsTmpl = template.Must(
	template.New("repo_plugins_go.tmpl").ParseFS(templates, "_templates/repo_plugins_go.tmpl"),
)

type autoloadImpl struct{}

// New returns this autoload stage for explicit registration in autoload/all.
func New() autoload.Autoload { return &autoloadImpl{} }

func (a *autoloadImpl) Name() string  { return "repos" }
func (a *autoloadImpl) Priority() int { return 300 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if ctx.Opts.Database == "" || ctx.ModPath == "" {
		return nil
	}

	pf, _ := polafile.Load(ctx.ProjectDir)
	repoDir := "repositories"
	entClientDir := "db/client/ent"
	if pf != nil {
		repoDir = pf.RepositoriesDir()
		entClientDir = pf.DatabaseEntClientDir()
	}

	repoDisco, err := discoverRepositoryRegistrations(ctx.ProjectDir, repoDir, entClientDir, ctx.Opts.Database, ctx.ModPath, ctx.Opts.PolaPackage)
	if err != nil {
		return err
	}
	if repoDisco == nil {
		return nil
	}

	ctx.Discovery.RepoDisco = repoDisco

	var buf strings.Builder
	if err := repoPluginsTmpl.Execute(&buf, repoDisco); err != nil {
		return fmt.Errorf("execute repo plugins template: %w", err)
	}

	repoPluginsPath := filepath.Join(ctx.TmpDir, "pola_repo_plugins.go")
	if err := os.WriteFile(repoPluginsPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write repo plugins: %w", err)
	}

	ormAbsDir, _ := filepath.Abs(filepath.Join(ctx.ProjectDir, repoDisco.RepoDir, ctx.Opts.Database))
	ctx.Replace[filepath.Join(ormAbsDir, "pola_plugins.go")] = repoPluginsPath

	return nil
}

type discoveredRepo struct {
	name     string
	filename string
	fd       *ast.FuncDecl
	file     *ast.File
}

// discoverRepositoryRegistrations scans repositories/{orm}/ for exported
// New*Repository constructors, extracts each constructor's parameters via
// ctorscan, and returns the merged plugin discovery data. Returns nil when
// no repositories are found; returns an error when a constructor uses an
// unsupported parameter shape.
func discoverRepositoryRegistrations(projectDir, repoDir, entClientDir, orm, modPath, polaPackage string) (*autoload.RepoDiscovery, error) {
	ormDir := filepath.Join(projectDir, repoDir, orm)
	info, err := os.Stat(ormDir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(ormDir)
	if err != nil {
		return nil, nil
	}

	var found []discoveredRepo
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(ormDir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || !fd.Name.IsExported() || fd.Recv != nil {
				continue
			}
			name := fd.Name.Name
			if strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Repository") {
				modelName := strings.TrimPrefix(name, "New")
				modelName = strings.TrimSuffix(modelName, "Repository")
				if modelName == "" {
					continue
				}
				found = append(found, discoveredRepo{name: modelName, filename: entry.Name(), fd: fd, file: f})
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
	var repos []autoload.PluginEntry
	var allParams []ctorscan.Param
	for _, d := range found {
		if _, dup := seen[d.name]; dup {
			continue
		}
		seen[d.name] = struct{}{}
		params, err := ctorscan.ScanParams(d.fd, d.file, d.filename, polaPackage)
		if err != nil {
			return nil, fmt.Errorf("autoload repos: %w", err)
		}
		allParams = append(allParams, params...)
		repos = append(repos, autoload.PluginEntry{
			Name:      d.name,
			SnakeName: schema.SnakeCase(d.name),
			Params:    params,
			Args:      renderArgs(params),
		})
	}

	skip := map[string]struct{}{polaPackage + "/core": {}}
	extra, _ := ctorscan.MergeImports(allParams, skip)

	return &autoload.RepoDiscovery{
		PolaPackage:  polaPackage,
		ImportPath:   modPath + "/" + filepath.ToSlash(filepath.Join(repoDir, orm)),
		RepoImport:   modPath + "/" + filepath.ToSlash(repoDir),
		ModulePath:   modPath,
		EntClientDir: entClientDir,
		RepoDir:      repoDir,
		ORM:          orm,
		PkgName:      orm,
		Repositories: repos,
		ExtraImports: extra,
	}, nil
}

// renderArgs joins the constructor arguments in the same order as the params
// slice. Registry params become "r"; others become "p0", "p1", ...
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

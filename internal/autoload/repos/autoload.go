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
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/polafile"
)

//go:embed _templates/repo_plugins_go.tmpl
var templates embed.FS

var repoPluginsTmpl = template.Must(
	template.New("repo_plugins_go.tmpl").ParseFS(templates, "_templates/repo_plugins_go.tmpl"),
)

type autoloadImpl struct{}

func init() {
	autoload.Register(&autoloadImpl{})
}

func (a *autoloadImpl) Name() string { return "repos" }
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

	repoDisco := discoverRepositoryRegistrations(ctx.ProjectDir, repoDir, entClientDir, ctx.Opts.Database, ctx.ModPath)
	if repoDisco == nil {
		return nil
	}

	ctx.Discovery.RepoDisco = repoDisco

	var buf strings.Builder
	if err := repoPluginsTmpl.Execute(&buf, struct {
		PolaPackage  string
		PkgName      string
		ORM          string
		RepoImport   string
		ModulePath   string
		EntClientDir string
		Repositories []autoload.PluginEntry
	}{
		PolaPackage:  ctx.Opts.PolaPackage,
		PkgName:      repoDisco.PkgName,
		ORM:          repoDisco.ORM,
		RepoImport:   repoDisco.RepoImport,
		ModulePath:   repoDisco.ModulePath,
		EntClientDir: repoDisco.EntClientDir,
		Repositories: repoDisco.Repositories,
	}); err != nil {
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

// discoverRepositoryRegistrations scans repositories/{orm}/ for exported
// New*Repository constructor functions and returns their names.
func discoverRepositoryRegistrations(projectDir, repoDir, entClientDir, orm, modPath string) *autoload.RepoDiscovery {
	ormDir := filepath.Join(projectDir, repoDir, orm)
	info, err := os.Stat(ormDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(ormDir)
	if err != nil {
		return nil
	}

	var repos []autoload.PluginEntry
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
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
				if modelName != "" {
					repos = append(repos, autoload.PluginEntry{
						Name:      modelName,
						SnakeName: schema.SnakeCase(modelName),
					})
				}
			}
		}
	}

	if len(repos) == 0 {
		return nil
	}

	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return &autoload.RepoDiscovery{
		ImportPath:   modPath + "/" + filepath.ToSlash(filepath.Join(repoDir, orm)),
		RepoImport:   modPath + "/" + filepath.ToSlash(repoDir),
		ModulePath:   modPath,
		EntClientDir: entClientDir,
		RepoDir:      repoDir,
		ORM:          orm,
		PkgName:      orm,
		Repositories: repos,
	}
}

// Package routes implements the route discovery and init overlay autoload.
package routes

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/autoload"
)

//go:embed _templates/route_init_go.tmpl
var templates embed.FS

var routeInitTmpl = template.Must(
	template.New("route_init_go.tmpl").ParseFS(templates, "_templates/route_init_go.tmpl"),
)

type autoloadImpl struct{}

func init() {
	autoload.Register(&autoloadImpl{})
}

func (a *autoloadImpl) Name() string { return "routes" }
func (a *autoloadImpl) Priority() int { return 200 }

func (a *autoloadImpl) Contribute(ctx *autoload.Context) error {
	if ctx.ModPath == "" {
		return nil
	}

	routePkgs := discoverRoutePackages(ctx.ProjectDir, ctx.ModPath)
	ctx.Discovery.RoutePkgs = routePkgs

	for i, rp := range routePkgs {
		dep := discoverRouteServiceDep(rp.AbsDir, ctx.ModPath)
		src, err := generateRouteInit(rp.PkgName, ctx.Opts.PolaPackage, ctx.ModPath, dep)
		if err != nil {
			return fmt.Errorf("generate route init for %s: %w", rp.PkgName, err)
		}
		initPath := filepath.Join(ctx.TmpDir, fmt.Sprintf("pola_route_init_%d.go", i))
		if err := os.WriteFile(initPath, src, 0o644); err != nil {
			return fmt.Errorf("write route init for %s: %w", rp.PkgName, err)
		}
		ctx.Replace[filepath.Join(rp.AbsDir, "pola_route_init.go")] = initPath
	}

	return nil
}

// discoverRoutePackages walks the routes/ directory and returns metadata
// for every sub-package that contains at least one .go file.
func discoverRoutePackages(projectDir, modPath string) []autoload.RoutePackageInfo {
	routesDir := filepath.Join(projectDir, "routes")
	info, err := os.Stat(routesDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var pkgs []autoload.RoutePackageInfo
	filepath.WalkDir(routesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				rel, _ := filepath.Rel(projectDir, path)
				absDir, _ := filepath.Abs(path)
				pkgs = append(pkgs, autoload.RoutePackageInfo{
					ImportPath: modPath + "/" + filepath.ToSlash(rel),
					AbsDir:     absDir,
					PkgName:    filepath.Base(path),
				})
				break
			}
		}
		return nil
	})

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].ImportPath < pkgs[j].ImportPath
	})
	return pkgs
}

// routeInitData drives the route_init_go.tmpl template. The discovery logic
// below derives the imports, factory body, and constructor args; the template
// only renders the file skeleton.
type routeInitData struct {
	PkgName string
	HasDep  bool
	Imports []string // raw import paths; the template quotes them
	Body    []string // factory body lines, e.g. "svc := core.MustInvoke[*pkg.T](r)"
	Args    string   // joined NewRoute arguments
}

// generateRouteInit returns the source for a pola_route_init.go file that
// registers a route factory via init().
func generateRouteInit(pkgName, polaPackage, modPath string, dep *autoload.RouteServiceDep) ([]byte, error) {
	imports := []string{
		polaPackage + "/core",
		polaPackage + "/routes",
	}
	data := routeInitData{PkgName: pkgName, Imports: imports}

	if dep != nil {
		data.HasDep = true
		var body, args []string
		if dep.ServicePkg != "" {
			imports = append(imports, dep.ServicePath)
			if dep.ServiceInterface {
				body = append(body, fmt.Sprintf("svc := core.MustInvoke[%s.%s](r)", dep.ServicePkg, dep.ServiceType))
			} else {
				body = append(body, fmt.Sprintf("svc := core.MustInvoke[*%s.%s](r)", dep.ServicePkg, dep.ServiceType))
			}
			args = append(args, "svc")
		}
		if dep.HasStorage {
			imports = append(imports, polaPackage+"/storage")
			body = append(body, "store := core.MustInvoke[storage.Storage](r)")
			args = append(args, "store")
		}
		if dep.BlobRepo != "" && modPath != "" {
			imports = append(imports, modPath+"/repositories")
			body = append(body, fmt.Sprintf("blobs := core.MustInvoke[repositories.%s](r)", dep.BlobRepo))
			args = append(args, "blobs")
		}
		data.Imports = imports
		data.Body = body
		data.Args = strings.Join(args, ", ")
	}

	var buf strings.Builder
	if err := routeInitTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute route init template: %w", err)
	}
	return []byte(buf.String()), nil
}

// discoverRouteServiceDep scans a route package for a Route struct with a field
// whose type matches *services.*Service, and returns the dependency info.
func discoverRouteServiceDep(routeDir, modPath string) *autoload.RouteServiceDep {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(routeDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if strings.HasPrefix(entry.Name(), "pola_") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(routeDir, entry.Name()), nil, 0)
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
				if !ok || ts.Name.Name != "Route" {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				dep := &autoload.RouteServiceDep{}
				for _, field := range st.Fields.List {
					switch t := field.Type.(type) {
					case *ast.StarExpr:
						sel, ok := t.X.(*ast.SelectorExpr)
						if !ok {
							continue
						}
						ident, ok := sel.X.(*ast.Ident)
						if !ok {
							continue
						}
						typeName := sel.Sel.Name
						if strings.HasSuffix(typeName, "Service") && dep.ServicePkg == "" {
							dep.ServicePkg = ident.Name
							dep.ServiceType = typeName
							dep.ServicePath = modPath + "/" + ident.Name
						}
					case *ast.SelectorExpr:
						ident, ok := t.X.(*ast.Ident)
						if !ok {
							continue
						}
						typeName := t.Sel.Name
						if ident.Name == "storage" && typeName == "Storage" {
							dep.HasStorage = true
						} else if ident.Name == "repositories" && strings.HasSuffix(typeName, "Repository") {
							dep.BlobRepo = typeName
						} else if strings.HasSuffix(typeName, "ServiceInterface") && dep.ServicePkg == "" {
							dep.ServicePkg = ident.Name
							dep.ServiceType = typeName
							dep.ServicePath = modPath + "/" + ident.Name
							dep.ServiceInterface = true
						}
					}
				}
				if dep.ServicePkg != "" || dep.HasStorage || dep.BlobRepo != "" {
					return dep
				}
			}
		}
	}
	return nil
}

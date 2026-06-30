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

	var routePkgs []autoload.RoutePackageInfo
	for _, rp := range discoverRoutePackages(ctx.ProjectDir, ctx.ModPath) {
		hasStruct, funcRoutes := scanRouteStyle(rp.AbsDir)
		// Skip directories under routes/ that aren't actually route packages
		// (no Route struct and no verb functions) — e.g. shared helpers.
		if !hasStruct && len(funcRoutes) == 0 {
			continue
		}
		routePkgs = append(routePkgs, rp)

		var dep *autoload.RouteServiceDep
		if hasStruct {
			dep = discoverRouteServiceDep(rp.AbsDir, ctx.ModPath)
		}
		src, err := generateRouteInit(rp.PkgName, ctx.Opts.PolaPackage, ctx.ModPath, dep, hasStruct, funcRoutes)
		if err != nil {
			return fmt.Errorf("generate route init for %s: %w", rp.PkgName, err)
		}
		initPath := filepath.Join(ctx.TmpDir, fmt.Sprintf("pola_route_init_%d.go", len(routePkgs)-1))
		if err := os.WriteFile(initPath, src, 0o644); err != nil {
			return fmt.Errorf("write route init for %s: %w", rp.PkgName, err)
		}
		ctx.Replace[filepath.Join(rp.AbsDir, "pola_route_init.go")] = initPath
	}
	ctx.Discovery.RoutePkgs = routePkgs

	return nil
}

// httpVerbs is the set of method names recognized as route handlers.
var httpVerbs = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	"HEAD": true, "OPTIONS": true, "CONNECT": true, "TRACE": true,
}

// scanRouteStyle inspects a route package's .go files (skipping generated
// pola_* files) and reports whether it declares a `Route` struct (struct-based)
// and which package-level functions are named after HTTP verbs (function-based).
func scanRouteStyle(routeDir string) (hasStruct bool, funcRoutes []string) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(routeDir)
	if err != nil {
		return false, nil
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasPrefix(entry.Name(), "pola_") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(routeDir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "Route" {
						hasStruct = true
					}
				}
			case *ast.FuncDecl:
				// Package-level function (no receiver) named after an HTTP verb,
				// with a single param and single result (func(core.Context) error).
				if d.Recv != nil || !httpVerbs[d.Name.Name] {
					continue
				}
				if d.Type.Params == nil || len(d.Type.Params.List) != 1 {
					continue
				}
				if d.Type.Results == nil || len(d.Type.Results.List) != 1 {
					continue
				}
				if !seen[d.Name.Name] {
					seen[d.Name.Name] = true
					funcRoutes = append(funcRoutes, d.Name.Name)
				}
			}
		}
	}
	sort.Strings(funcRoutes)
	return hasStruct, funcRoutes
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
	PkgName    string
	HasStruct  bool     // package declares a Route struct (struct-based)
	HasDep     bool     // Route struct has constructor dependencies
	Imports    []string // raw import paths; the template quotes them
	Body       []string // factory body lines, e.g. "svc := core.MustInvoke[*pkg.T](r)"
	Args       string   // joined NewRoute arguments
	FuncRoutes []string // verb names of package-level function routes
}

// generateRouteInit returns the source for a pola_route_init.go file that
// registers struct-based and/or function-based routes via init().
func generateRouteInit(pkgName, polaPackage, modPath string, dep *autoload.RouteServiceDep, hasStruct bool, funcRoutes []string) ([]byte, error) {
	imports := []string{polaPackage + "/routes"}
	if hasStruct {
		imports = append(imports, polaPackage+"/core")
	}
	data := routeInitData{PkgName: pkgName, HasStruct: hasStruct, FuncRoutes: funcRoutes, Imports: imports}

	if dep != nil {
		data.HasDep = true
		var body, args []string
		if dep.ServicePkg != "" {
			imports = append(imports, dep.ServicePath)
			body = append(body, fmt.Sprintf("svc := core.MustInvoke[*%s.%s](r)", dep.ServicePkg, dep.ServiceType))
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

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
	"github.com/polagonow/pola/internal/autoload/ctorscan"
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

func (a *autoloadImpl) Name() string  { return "routes" }
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

		var ctorParams []ctorscan.Param
		var hasCtor bool
		var dep *autoload.RouteServiceDep
		if hasStruct {
			found, p, err := discoverRouteCtor(rp.AbsDir, ctx.Opts.PolaPackage)
			if err != nil {
				return fmt.Errorf("scan NewRoute in %s: %w", rp.PkgName, err)
			}
			if found {
				hasCtor = true
				ctorParams = p
			} else {
				dep = discoverRouteServiceDep(rp.AbsDir, ctx.ModPath)
			}
		}
		src, err := generateRouteInit(rp.PkgName, ctx.Opts.PolaPackage, ctx.ModPath, dep, hasStruct, funcRoutes, hasCtor, ctorParams)
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

// discoverRouteCtor scans a route package for a package-level func NewRoute(...)
// and returns the ctorscan-resolved parameters. The first return value is true
// when a NewRoute constructor exists — including the zero-parameter case where
// the returned param slice is nil/empty. Returns (false, nil, nil) when no
// NewRoute exists — callers fall back to field-scan; returns an error when a
// parameter type cannot be resolved.
func discoverRouteCtor(routeDir, polaPackage string) (bool, []ctorscan.Param, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(routeDir)
	if err != nil {
		return false, nil, nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasPrefix(entry.Name(), "pola_") {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(routeDir, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || fd.Name.Name != "NewRoute" {
				continue
			}
			params, err := ctorscan.ScanParams(fd, f, entry.Name(), polaPackage)
			return true, params, err
		}
	}
	return false, nil, nil
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

// routeInitData drives the route_init_go.tmpl template.
type routeInitData struct {
	PkgName    string
	HasStruct  bool              // package declares a Route struct
	HasDep     bool              // Route struct has constructor dependencies
	Imports    []ctorscan.Import // alias-aware imports
	Body       []string          // factory body lines
	Args       string            // joined NewRoute arguments
	FuncRoutes []string          // verb names of package-level function routes
}

// generateRouteInit returns the source for a pola_route_init.go file that
// registers struct-based and/or function-based routes via init().
//
// When hasCtor is true, args and body are derived from the NewRoute
// constructor parameters (registry-style; zero params → NewRoute()).
// When hasCtor is false, the legacy field-scan dep is used.
func generateRouteInit(pkgName, polaPackage, modPath string, dep *autoload.RouteServiceDep, hasStruct bool, funcRoutes []string, hasCtor bool, ctorParams []ctorscan.Param) ([]byte, error) {
	imports := []ctorscan.Import{{Path: polaPackage + "/routes"}}
	if hasStruct {
		imports = append(imports, ctorscan.Import{Path: polaPackage + "/core"})
	}
	data := routeInitData{PkgName: pkgName, HasStruct: hasStruct, FuncRoutes: funcRoutes}

	if hasStruct && hasCtor {
		data.HasDep = true
		var body, args []string
		for i, p := range ctorParams {
			if p.IsRegistry {
				args = append(args, "r")
				continue
			}
			local := fmt.Sprintf("p%d", i)
			body = append(body, fmt.Sprintf("%s := core.MustInvoke[%s](r)", local, p.Type))
			args = append(args, local)
		}
		data.Body = body
		data.Args = strings.Join(args, ", ")
		skip := map[string]struct{}{polaPackage + "/core": {}}
		extra, _ := ctorscan.MergeImports(ctorParams, skip)
		imports = append(imports, extra...)
	} else if dep != nil {
		data.HasDep = true
		var body, args []string
		if dep.ServicePkg != "" {
			imports = append(imports, ctorscan.Import{Path: dep.ServicePath})
			if dep.ServiceInterface {
				body = append(body, fmt.Sprintf("svc := core.MustInvoke[%s.%s](r)", dep.ServicePkg, dep.ServiceType))
			} else {
				body = append(body, fmt.Sprintf("svc := core.MustInvoke[*%s.%s](r)", dep.ServicePkg, dep.ServiceType))
			}
			args = append(args, "svc")
		}
		if dep.HasStorage {
			imports = append(imports, ctorscan.Import{Path: polaPackage + "/storage"})
			body = append(body, "store := core.MustInvoke[storage.Storage](r)")
			args = append(args, "store")
		}
		if dep.BlobRepo != "" && modPath != "" {
			imports = append(imports, ctorscan.Import{Path: modPath + "/repositories"})
			body = append(body, fmt.Sprintf("blobs := core.MustInvoke[repositories.%s](r)", dep.BlobRepo))
			args = append(args, "blobs")
		}
		data.Body = body
		data.Args = strings.Join(args, ", ")
	}
	data.Imports = imports

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

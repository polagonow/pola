package cli

import (
	"embed"
	"encoding/json"
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

	"github.com/polagonow/pola/internal/actionbridge"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/polafile"
)

//go:embed _templates/plugins_go.tmpl _templates/embed_go.tmpl _templates/repo_plugins_go.tmpl _templates/svc_plugins_go.tmpl
var overlayTemplates embed.FS

var pluginsTmpl = template.Must(
	template.New("plugins_go.tmpl").ParseFS(overlayTemplates, "_templates/plugins_go.tmpl"),
)

var repoPluginsTmpl = template.Must(
	template.New("repo_plugins_go.tmpl").ParseFS(overlayTemplates, "_templates/repo_plugins_go.tmpl"),
)

var svcPluginsTmpl = template.Must(
	template.New("svc_plugins_go.tmpl").ParseFS(overlayTemplates, "_templates/svc_plugins_go.tmpl"),
)

// embedTmpl is the template for pola_embed.go, injected via overlay
// during production builds. It embeds the public/ directory and registers the
// asset server and prebuild loader as a plugin.
var embedTmpl = template.Must(
	template.New("embed_go.tmpl").ParseFS(overlayTemplates, "_templates/embed_go.tmpl"),
)

// pluginOpts holds parameters for plugin generation.
type pluginOpts struct {
	PolaPackage     string
	Engine          string
	Bundler         string
	Renderer        string
	Router          string
	CSS             string
	Cache           string
	Database        string // ORM name: "ent", "gorm", "beego", or "" for none.
	DatabaseAdapter string // "postgresql", "mysql", "sqlite"
	DatabaseURL     string
	DatabaseHost    string
	DatabasePort    string
	DatabaseUser    string
	DatabasePass    string
	DatabaseName    string
	CSRF            bool
	SecurityHeaders bool
	Dev             bool
	Embed           bool
}

// overlayResult holds the output from generateOverlay.
type overlayResult struct {
	OverlayPath string
	TmpDir      string
	TSOutPath   string
}

// routePackageInfo holds metadata about a discovered route package.
type routePackageInfo struct {
	ImportPath string // e.g. "test-app/routes/kampala/uganda"
	AbsDir     string // e.g. "/abs/path/routes/kampala/uganda"
	PkgName    string // e.g. "uganda"
}

// generateOverlay creates a unified overlay containing:
//  1. pola_plugins.go — explicit Plugin() calls (always)
//  2. generated_bridge.go — action bridge codegen (if actions/ exists)
//  3. pola_embed.go — asset embedding for production builds (//go:build embed)
//
// The caller should defer os.RemoveAll(result.TmpDir) after the build completes.
func generateOverlay(projectDir string, opts pluginOpts) (*overlayResult, error) {
	tmpDir, err := os.MkdirTemp("", "pola-overlay-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	replace := make(map[string]string)

	// Read the module path once — used by multiple discovery steps below.
	modPath, _ := readModulePath(projectDir)

	// Determine actions directory.
	actionsDir := generateFlags.actionsDir
	if actionsDir == "" {
		actionsDir = filepath.Join(projectDir, "actions")
	}
	if !filepath.IsAbs(actionsDir) {
		actionsDir = filepath.Join(projectDir, actionsDir)
	}

	// 1. Run action bridge codegen if actions/ exists.
	var tsOutPath string
	var actionsImport string
	hasActions := false
	if info, err := os.Stat(actionsDir); err == nil && info.IsDir() {
		hasActions = true
		tsOut := generateFlags.tsOut
		if tsOut == "" {
			tsOut = filepath.Join(projectDir, "node_modules", "@pola", "actions", "src", "generated.ts")
		}
		if !filepath.IsAbs(tsOut) {
			tsOut = filepath.Join(projectDir, tsOut)
		}

		fmt.Println("Generating action bridges...")
		bridgeResult, err := actionbridge.Run(actionsDir, tsOut, tmpDir, opts.PolaPackage)
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("actionbridge: %w", err)
		}

		if bridgeResult != nil && bridgeResult.BridgePath != "" {
			replace[bridgeResult.VirtualPath] = bridgeResult.BridgePath
			tsOutPath = bridgeResult.TSOutPath
		}

		if verbose && bridgeResult != nil && bridgeResult.TSOutPath != "" {
			fmt.Printf("Generated types: %s\n", bridgeResult.TSOutPath)
		}
	} else if verbose {
		fmt.Println("No actions/ directory found, skipping actionbridge.")
	}

	// 2. Resolve the actions import path so pola_plugins.go can blank-import it.
	if hasActions && modPath != "" {
		actionsImport = modPath + "/actions"
	}

	// 3. Discover route packages under routes/.
	var routePkgs []routePackageInfo
	if modPath != "" {
		routePkgs = discoverRoutePackages(projectDir, modPath)
	}

	// 3b. Generate pola_route_init.go overlay for each route package.
	//     If the route has a service dependency, the factory calls NewRoute(svc).
	for i, rp := range routePkgs {
		dep := discoverRouteServiceDep(rp.AbsDir, modPath)
		src := generateRouteInit(rp.PkgName, opts.PolaPackage, dep)
		initPath := filepath.Join(tmpDir, fmt.Sprintf("pola_route_init_%d.go", i))
		if err := os.WriteFile(initPath, src, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("write route init for %s: %w", rp.PkgName, err)
		}
		replace[filepath.Join(rp.AbsDir, "pola_route_init.go")] = initPath
	}

	// 3c. Discover repository registrations and generate per-repo plugin overlay.
	var repoDisco *repoDiscovery
	if opts.Database != "" && modPath != "" {
		pf, _ := polafile.Load(projectDir)
		repoDir := "repositories"
		if pf != nil {
			repoDir = pf.RepositoriesDir()
		}
		repoDisco = discoverRepositoryRegistrations(projectDir, repoDir, opts.Database, modPath)
	}

	if repoDisco != nil {
		var repoPluginsBuf strings.Builder
		if err := repoPluginsTmpl.Execute(&repoPluginsBuf, struct {
			PolaPackage  string
			PkgName      string
			ORM          string
			RepoImport   string
			ModulePath   string
			Repositories []pluginEntry
		}{
			PolaPackage:  opts.PolaPackage,
			PkgName:      repoDisco.PkgName,
			ORM:          repoDisco.ORM,
			RepoImport:   repoDisco.RepoImport,
			ModulePath:   repoDisco.ModulePath,
			Repositories: repoDisco.Repositories,
		}); err != nil {
			return nil, fmt.Errorf("execute repo plugins template: %w", err)
		}
		repoPluginsPath := filepath.Join(tmpDir, "pola_repo_plugins.go")
		if err := os.WriteFile(repoPluginsPath, []byte(repoPluginsBuf.String()), 0o644); err != nil {
			return nil, fmt.Errorf("write repo plugins: %w", err)
		}
		// Map into the ORM package directory.
		pf, _ := polafile.Load(projectDir)
		repoDir := "repositories"
		if pf != nil {
			repoDir = pf.RepositoriesDir()
		}
		ormAbsDir, _ := filepath.Abs(filepath.Join(projectDir, repoDir, opts.Database))
		replace[filepath.Join(ormAbsDir, "pola_plugins.go")] = repoPluginsPath
	}

	// 3d. Discover service constructors and generate per-service plugin overlay.
	var svcDisco *svcDiscovery
	if modPath != "" {
		pf, _ := polafile.Load(projectDir)
		svcDir := "services"
		if pf != nil {
			svcDir = pf.ServicesDir()
		}
		svcDisco = discoverServiceConstructors(projectDir, svcDir, modPath)
	}

	if svcDisco != nil {
		var svcPluginsBuf strings.Builder
		if err := svcPluginsTmpl.Execute(&svcPluginsBuf, struct {
			PolaPackage string
			PkgName     string
			RepoImport  string
			Services    []pluginEntry
		}{
			PolaPackage: opts.PolaPackage,
			PkgName:     svcDisco.PkgName,
			RepoImport:  svcDisco.RepoImport,
			Services:    svcDisco.Services,
		}); err != nil {
			return nil, fmt.Errorf("execute svc plugins template: %w", err)
		}
		svcPluginsPath := filepath.Join(tmpDir, "pola_svc_plugins.go")
		if err := os.WriteFile(svcPluginsPath, []byte(svcPluginsBuf.String()), 0o644); err != nil {
			return nil, fmt.Errorf("write svc plugins: %w", err)
		}
		svcAbsDir, _ := filepath.Abs(filepath.Join(projectDir, svcDisco.PkgName))
		replace[filepath.Join(svcAbsDir, "pola_plugins.go")] = svcPluginsPath
	}

	// 4. Generate plugin imports (always).
	pluginsSrc, err := generatePluginImports(opts, actionsImport, routePkgs, repoDisco, svcDisco)
	if err != nil {
		return nil, fmt.Errorf("generate plugins: %w", err)
	}
	pluginsPath := filepath.Join(tmpDir, "pola_plugins.go")
	if err := os.WriteFile(pluginsPath, pluginsSrc, 0o644); err != nil {
		return nil, fmt.Errorf("write plugins: %w", err)
	}
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("abs project dir: %w", err)
	}
	replace[filepath.Join(absProjectDir, "pola_plugins.go")] = pluginsPath

	// 5. Generate embed file (only for production embed builds).
	if opts.Embed {
		var embedBuf strings.Builder
		if err := embedTmpl.Execute(&embedBuf, struct{ PolaPackage string }{opts.PolaPackage}); err != nil {
			return nil, fmt.Errorf("execute embed template: %w", err)
		}
		embedPath := filepath.Join(tmpDir, "pola_embed.go")
		if err := os.WriteFile(embedPath, []byte(embedBuf.String()), 0o644); err != nil {
			return nil, fmt.Errorf("write embed: %w", err)
		}
		replace[filepath.Join(absProjectDir, "pola_embed.go")] = embedPath
	}

	// 6. Write unified overlay JSON.
	overlay := map[string]any{
		"Replace": replace,
	}
	overlayJSON, err := json.Marshal(overlay)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("marshal overlay: %w", err)
	}

	overlayPath := filepath.Join(tmpDir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlayJSON, 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("write overlay: %w", err)
	}

	if verbose {
		fmt.Printf("Generated overlay: %s\n", overlayPath)
	}

	return &overlayResult{
		OverlayPath: overlayPath,
		TmpDir:      tmpDir,
		TSOutPath:   tsOutPath,
	}, nil
}

// generatePluginImports returns the source for pola_plugins.go containing
// explicit Plugin() calls and a PolaPlugins variable.
func generatePluginImports(opts pluginOpts, actionsImport string, routePkgs []routePackageInfo, repoDisco *repoDiscovery, svcDisco *svcDiscovery) ([]byte, error) {
	routeImports := make([]string, len(routePkgs))
	for i, rp := range routePkgs {
		routeImports[i] = rp.ImportPath
	}

	hasCSS := opts.CSS != "" && opts.CSS != "none"
	hasCache := opts.Cache != "" && opts.Cache != "none"
	hasDatabase := opts.Database != ""
	hasCSRF := opts.CSRF
	hasSecurityHeaders := opts.SecurityHeaders

	var buf strings.Builder
	err := pluginsTmpl.Execute(&buf, struct {
		PolaPackage     string
		Engine          string
		Bundler         string
		Renderer        string
		Router          string
		CSS             string
		Cache           string
		Database        string
		DatabaseAdapter string
		DatabaseURL     string
		DatabaseHost    string
		DatabasePort    string
		DatabaseUser    string
		DatabasePass    string
		DatabaseName    string
		CSRF            bool
		SecurityHeaders bool
		Dev             bool
		Embed           bool
		HasRoutes        bool
		ActionsImport    string
		RouteImports     []string
		RepoPlugins      *repoDiscovery
		ServicePlugins   *svcDiscovery
	}{
		PolaPackage:      opts.PolaPackage,
		Engine:           opts.Engine,
		Bundler:          opts.Bundler,
		Renderer:         opts.Renderer,
		Router:           opts.Router,
		CSS:              condStr(hasCSS, opts.CSS, ""),
		Cache:            condStr(hasCache, opts.Cache, ""),
		Database:         condStr(hasDatabase, opts.Database, ""),
		DatabaseAdapter:  opts.DatabaseAdapter,
		DatabaseURL:      opts.DatabaseURL,
		DatabaseHost:     opts.DatabaseHost,
		DatabasePort:     opts.DatabasePort,
		DatabaseUser:     opts.DatabaseUser,
		DatabasePass:     opts.DatabasePass,
		DatabaseName:     opts.DatabaseName,
		CSRF:             hasCSRF,
		SecurityHeaders:  hasSecurityHeaders,
		Dev:              opts.Dev,
		Embed:            opts.Embed,
		HasRoutes:        len(routePkgs) > 0,
		ActionsImport:    actionsImport,
		RouteImports:     routeImports,
		RepoPlugins:      repoDisco,
		ServicePlugins:   svcDisco,
	})
	if err != nil {
		return nil, fmt.Errorf("execute plugins template: %w", err)
	}
	return []byte(buf.String()), nil
}

// envOrBool returns the boolean value from an env var if set, otherwise the fallback.
func envOrBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1"
}

// databaseORM returns the configured ORM name if a database block exists, or "" otherwise.
func databaseORM(pf *polafile.Polafile) string {
	if pf == nil || pf.Database == nil {
		return ""
	}
	return pf.DatabaseORM()
}

// populateDatabaseOpts fills database-related fields in pluginOpts from the Polafile.
func populateDatabaseOpts(opts *pluginOpts, pf *polafile.Polafile, env string) {
	opts.Database = databaseORM(pf)
	if opts.Database == "" {
		return
	}
	merged := pf.DatabaseForEnv(env)
	opts.DatabaseAdapter = pf.DatabaseAdapter(env)
	opts.DatabaseURL = merged.URL
	opts.DatabaseHost = merged.Host
	opts.DatabasePort = merged.Port
	opts.DatabaseUser = merged.User
	opts.DatabasePass = merged.Password
	opts.DatabaseName = merged.Name
}

func condStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// discoverRoutePackages walks the routes/ directory and returns metadata
// for every sub-package that contains at least one .go file.
func discoverRoutePackages(projectDir, modPath string) []routePackageInfo {
	routesDir := filepath.Join(projectDir, "routes")
	info, err := os.Stat(routesDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	var pkgs []routePackageInfo
	filepath.WalkDir(routesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		// Check if this directory has any .go files.
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				rel, _ := filepath.Rel(projectDir, path)
				absDir, _ := filepath.Abs(path)
				pkgs = append(pkgs, routePackageInfo{
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

// generateRouteInit returns the source for a pola_route_init.go file that
// registers a route factory via init(). If dep is non-nil, the factory
// resolves the service via DI and calls NewRoute(svc); otherwise it returns &Route{}.
func generateRouteInit(pkgName, polaPackage string, dep *routeServiceDep) []byte {
	if dep == nil {
		return []byte(fmt.Sprintf(
			"// Code generated by pola; DO NOT EDIT.\npackage %s\n\nimport (\n\t\"%s/core\"\n\t\"%s/routes\"\n)\n\nfunc init() {\n\troutes.Register(func(_ *core.Registry) any { return &Route{} })\n}\n",
			pkgName, polaPackage, polaPackage,
		))
	}
	return []byte(fmt.Sprintf(
		"// Code generated by pola; DO NOT EDIT.\npackage %s\n\nimport (\n\t\"%s/core\"\n\t\"%s/routes\"\n\t\"%s\"\n)\n\nfunc init() {\n\troutes.Register(func(r *core.Registry) any {\n\t\tsvc := core.MustInvoke[*%s.%s](r)\n\t\treturn NewRoute(svc)\n\t})\n}\n",
		pkgName, polaPackage, polaPackage, dep.ServicePath, dep.ServicePkg, dep.ServiceType,
	))
}

// pluginEntry holds a discovered plugin name with its pre-computed snake_case form.
type pluginEntry struct {
	Name      string // e.g. "User"
	SnakeName string // e.g. "user"
}

// repoDiscovery holds discovered repository registration info for the overlay.
type repoDiscovery struct {
	ImportPath   string        // e.g. "myapp/repositories/gorm"
	RepoImport   string        // e.g. "myapp/repositories"
	ModulePath   string        // e.g. "myapp"
	ORM          string        // e.g. "gorm", "ent", "beego"
	PkgName      string        // e.g. "gorm"
	Repositories []pluginEntry // e.g. [{Name: "User", SnakeName: "user"}]
}

// svcDiscovery holds discovered service constructor info for the overlay.
type svcDiscovery struct {
	ImportPath string        // e.g. "myapp/services"
	RepoImport string        // e.g. "myapp/repositories"
	PkgName    string        // e.g. "services"
	Services   []pluginEntry // e.g. [{Name: "User", SnakeName: "user"}]
}

// discoverRepositoryRegistrations scans repositories/{orm}/ for exported
// New*Repository constructor functions and returns their names.
func discoverRepositoryRegistrations(projectDir, repoDir, orm, modPath string) *repoDiscovery {
	ormDir := filepath.Join(projectDir, repoDir, orm)
	info, err := os.Stat(ormDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(ormDir)
	if err != nil {
		return nil
	}

	var repos []pluginEntry
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
				// Extract the model name: New{Name}Repository -> Name
				modelName := strings.TrimPrefix(name, "New")
				modelName = strings.TrimSuffix(modelName, "Repository")
				if modelName != "" {
					repos = append(repos, pluginEntry{
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
	return &repoDiscovery{
		ImportPath:   modPath + "/" + filepath.ToSlash(filepath.Join(repoDir, orm)),
		RepoImport:   modPath + "/" + filepath.ToSlash(repoDir),
		ModulePath:   modPath,
		ORM:          orm,
		PkgName:      orm,
		Repositories: repos,
	}
}

// discoverServiceConstructors scans the services directory for exported
// New*Service constructor functions and returns their names.
func discoverServiceConstructors(projectDir, svcDir, modPath string) *svcDiscovery {
	absDir := filepath.Join(projectDir, svcDir)
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil
	}

	var svcs []pluginEntry
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
				// Extract the model name: New{Name}Service -> Name
				svcName := strings.TrimPrefix(name, "New")
				svcName = strings.TrimSuffix(svcName, "Service")
				if svcName != "" {
					svcs = append(svcs, pluginEntry{
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

	// Determine the repositories import path (sibling to services dir).
	pf, _ := polafile.Load(projectDir)
	repoDir := "repositories"
	if pf != nil {
		repoDir = pf.RepositoriesDir()
	}

	sort.Slice(svcs, func(i, j int) bool { return svcs[i].Name < svcs[j].Name })
	return &svcDiscovery{
		ImportPath: modPath + "/" + filepath.ToSlash(svcDir),
		RepoImport: modPath + "/" + filepath.ToSlash(repoDir),
		PkgName:    filepath.Base(svcDir),
		Services:   svcs,
	}
}

// routeServiceDep holds the service dependency info discovered in a route package.
type routeServiceDep struct {
	ServicePkg  string // e.g. "services"
	ServiceType string // e.g. "PostService"
	ServicePath string // e.g. "myapp/services"
}

// discoverRouteServiceDep scans a route package for a Route struct with a field
// whose type matches *services.*Service, and returns the dependency info.
func discoverRouteServiceDep(routeDir, modPath string) *routeServiceDep {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(routeDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		// Skip generated overlay files.
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
				for _, field := range st.Fields.List {
					// Look for *services.XxxService fields.
					pt, ok := field.Type.(*ast.StarExpr)
					if !ok {
						continue
					}
					sel, ok := pt.X.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					ident, ok := sel.X.(*ast.Ident)
					if !ok {
						continue
					}
					typeName := sel.Sel.Name
					if strings.HasSuffix(typeName, "Service") {
						return &routeServiceDep{
							ServicePkg:  ident.Name,
							ServiceType: typeName,
							ServicePath: modPath + "/" + ident.Name,
						}
					}
				}
			}
		}
	}
	return nil
}

// readModulePath reads the module path from go.mod in the given directory.
func readModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

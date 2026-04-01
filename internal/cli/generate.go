package cli

import (
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/polagonow/pola/internal/codegen"
	"github.com/polagonow/pola/internal/modelgen"
	_ "github.com/polagonow/pola/internal/modelgen/plugins" // register model generators
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var generateTemplates embed.FS

var actionTmpl = template.Must(
	template.New("action_go.tmpl").Delims("[[", "]]").ParseFS(generateTemplates, "_templates/action_go.tmpl"),
)

var routeTmpl = template.Must(
	template.New("route_go.tmpl").Delims("[[", "]]").ParseFS(generateTemplates, "_templates/route_go.tmpl"),
)

var pluginsTmpl = template.Must(
	template.New("plugins_go.tmpl").ParseFS(generateTemplates, "_templates/plugins_go.tmpl"),
)

// embedTmpl is the template for pola_embed.go, injected via overlay
// during production builds. It embeds the public/ directory and registers the
// asset server and prebuild loader as a plugin.
var embedTmpl = template.Must(
	template.New("embed_go.tmpl").ParseFS(generateTemplates, "_templates/embed_go.tmpl"),
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
	CSRF            bool
	SecurityHeaders bool
	Dev             bool
	Embed           bool
}

var generateFlags struct {
	actionsDir string
	tsOut      string
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate bridge code from actions/ directory",
	Long: `Parse Go structs in the actions/ directory and generate:
  - Go RuntimeInjector wiring (injected via -overlay, not written to your project)
  - TypeScript declaration file (typed di imports)`,
	RunE: runGenerate,
	Example: `  pola generate
  pola generate --actions-dir ./actions --ts-out ./ui/packages/di/src/generated.d.ts`,
	Aliases: []string{"gen", "g"},
}

var generateActionCmd = &cobra.Command{
	Use:   "action [Name]",
	Short: "Scaffold a new action struct",
	Long:  "Create a new action file in the actions/ directory with boilerplate and comments.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateAction,
	Example: `  pola generate action Blog
  pola generate action Products`,
}

var generateRouteCmd = &cobra.Command{
	Use:   "route [Name]",
	Short: "Scaffold a new route handler",
	Long:  "Create a new route file in the routes/ directory with HTTP method stubs.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateRoute,
	Example: `  pola generate route Posts
  pola generate route Posts --actions GET,POST
  pola generate route Posts/Comments --actions GET,POST,DELETE`,
}

var generateModelCmd = &cobra.Command{
	Use:   "model [Name] [field:type ...]",
	Short: "Generate an ORM model/schema from field definitions",
	Long: `Parse model name and field:type{opts}:modifier specs, then generate
ORM-specific schema files using the plugin configured in Polafile.hcl.

Field syntax: field:type{options}:modifier1:modifier2
  Types:    string, int, int64, float, bool, time, uuid, text, bytes, json, references
  Options:  {polymorphic} (only on references)
  Modifiers: index, uniq`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    runGenerateModel,
	Aliases: []string{"m"},
	Example: `  pola generate model User name:string email:string:uniq age:int
  pola generate model Article title:string:index body:text author:references
  pola generate model Comment body:text commentable:references{polymorphic}`,
}

var generateForce bool
var generateSkipCollision bool
var routeActions string

// validHTTPMethods is the set of HTTP methods accepted by --actions.
var validHTTPMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	"HEAD": true, "OPTIONS": true, "CONNECT": true, "TRACE": true,
}

func init() {
	generateCmd.Flags().StringVar(&generateFlags.actionsDir, "actions-dir", "", "path to actions directory (default: ./actions)")
	generateCmd.Flags().StringVar(&generateFlags.tsOut, "ts-out", "", "path to generated .d.ts file")
	generateCmd.PersistentFlags().BoolVar(&generateForce, "force", false, "overwrite files that already exist")
	generateCmd.PersistentFlags().BoolVar(&generateSkipCollision, "skip-collision-check", false, "skip collision check entirely")
	generateRouteCmd.Flags().StringVar(&routeActions, "actions", "GET", "comma-separated HTTP methods (e.g. GET,POST,DELETE)")
	generateCmd.AddCommand(generateActionCmd)
	generateCmd.AddCommand(generateRouteCmd)
	generateCmd.AddCommand(generateModelCmd)
}

func runGenerate(_ *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	// Load Polafile defaults (env vars take precedence, then Polafile, then hardcoded).
	var pf polafile.Polafile
	if loaded, err := polafile.Load(projectDir); err == nil && loaded != nil {
		pf = *loaded
	}

	result, err := generateOverlay(projectDir, pluginOpts{
		PolaPackage:     pf.PolaPackage(),
		Engine:          cmp.Or(os.Getenv("POLA_VM"), nameOnly(pf.Engine), "goja"),
		Bundler:         cmp.Or(os.Getenv("POLA_BUNDLER"), nameOnly(pf.Bundler), "esbuild"),
		Renderer:        cmp.Or(os.Getenv("POLA_RENDERER"), nameOnly(pf.Renderer), "react"),
		Router:          cmp.Or(os.Getenv("POLA_ROUTER"), nameOnly(pf.Router), "nextjs"),
		CSS:             cmp.Or(os.Getenv("POLA_CSS"), nameOnly(pf.CSS), "tailwind"),
		Cache:           cmp.Or(os.Getenv("POLA_CACHE"), pf.CacheAdapter("default"), "memory"),
		CSRF:            envOrBool("POLA_CSRF", pf.CSRFEnabled("default")),
		SecurityHeaders: envOrBool("POLA_SECURITY_HEADERS", pf.SecurityHeadersEnabled("default")),
		Dev:             true,
	})
	if err != nil {
		return err
	}
	if result != nil && result.TmpDir != "" {
		defer os.RemoveAll(result.TmpDir)
	}
	if result != nil && result.OverlayPath != "" {
		fmt.Printf("Overlay: %s\n", result.OverlayPath)
		fmt.Println("Pass -overlay to go build/run to include the generated bridge.")
	}
	return nil
}

func runGenerateAction(_ *cobra.Command, args []string) error {
	name := args[0]
	if name[0] >= 'a' && name[0] <= 'z' {
		name = string(name[0]-32) + name[1:]
	}

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(projectDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		return fmt.Errorf("create actions dir: %w", err)
	}

	filename := strings.ToLower(name) + ".go"
	filePath := filepath.Join(actionsDir, filename)

	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("%s already exists", filePath)
	}

	var buf strings.Builder
	if err := actionTmpl.Execute(&buf, struct{ Name string }{Name: name}); err != nil {
		return fmt.Errorf("execute action template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)
	return nil
}

func runGenerateRoute(_ *cobra.Command, args []string) error {
	name := args[0]

	// Split on "/" for nested routes: Posts/Comments → ["posts", "comments"].
	segments := strings.Split(name, "/")
	for i, s := range segments {
		segments[i] = strings.ToLower(s)
	}
	pkgName := segments[len(segments)-1]

	// Parse and validate HTTP methods.
	methods, err := parseActions(routeActions)
	if err != nil {
		return err
	}

	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(append([]string{projectDir, "routes"}, segments...)...)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create route dir: %w", err)
	}

	filePath := filepath.Join(targetDir, "route.go")
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("%s already exists", filePath)
	}

	var buf strings.Builder
	if err := routeTmpl.Execute(&buf, struct {
		Package string
		Methods []string
	}{
		Package: pkgName,
		Methods: methods,
	}); err != nil {
		return fmt.Errorf("execute route template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)
	return nil
}

func runGenerateModel(_ *cobra.Command, args []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	// Load Polafile.
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		pf = &polafile.Polafile{}
	}

	// Ensure database block exists; prompt interactively for missing ORM.
	if pf.Database == nil {
		pf.Database = &polafile.Database{
			Models:     "models",
			Migrations: "migrations",
		}
	}
	dirty := false
	if pf.Database.ORM == "" {
		orm, err := promptSelect("ORM:", []string{"ent", "gorm"})
		if err != nil {
			return err
		}
		pf.Database.ORM = orm
		dirty = true
	}
	if dirty {
		if err := polafile.Save(projectDir, pf); err != nil {
			return fmt.Errorf("save Polafile: %w", err)
		}
		fmt.Println("Saved database configuration to Polafile.hcl.")
	}

	modelDef, err := modelgen.ParseArgs(args)
	if err != nil {
		return err
	}

	orm := pf.DatabaseORM()
	outDir := filepath.Join(projectDir, pf.DatabaseModelsDir())

	// Validate that referenced models exist and resolve FK types.
	if err := modelgen.ValidateReferences(modelDef, outDir, orm); err != nil {
		return err
	}

	gen, err := modelgen.GetGenerator(orm)
	if err != nil {
		return err
	}

	outFile := filepath.Join(outDir, orm, modelgen.SnakeCase(modelDef.Name)+".go")

	// Collision check: if file exists, respect --force / --skip-collision-check.
	if !generateSkipCollision {
		if _, err := os.Stat(outFile); err == nil {
			if !generateForce {
				return fmt.Errorf("%s already exists; use --force to overwrite or --skip-collision-check to skip", outFile)
			}
			fmt.Printf("Overwriting %s\n", outFile)
		}
	}

	if err := gen.Generate(modelDef, outDir); err != nil {
		return fmt.Errorf("generate model: %w", err)
	}

	fmt.Printf("Created %s\n", outFile)
	return nil
}

// promptSelect presents an interactive selection prompt and returns the chosen option.
func promptSelect(message string, options []string) (string, error) {
	var answer string
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}
	return answer, nil
}

// parseActions splits and validates the --actions flag value.
func parseActions(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	seen := make(map[string]bool, len(parts))
	var methods []string
	for _, p := range parts {
		m := strings.TrimSpace(strings.ToUpper(p))
		if m == "" {
			continue
		}
		if !validHTTPMethods[m] {
			return nil, fmt.Errorf("unknown HTTP method %q; valid methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, CONNECT, TRACE", m)
		}
		if seen[m] {
			continue
		}
		seen[m] = true
		methods = append(methods, m)
	}
	if len(methods) == 0 {
		methods = []string{"GET"}
	}
	return methods, nil
}

// overlayResult holds the output from generateOverlay.
type overlayResult struct {
	OverlayPath string
	TmpDir      string
	TSOutPath   string
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
		codegenResult, err := codegen.Run(actionsDir, tsOut, tmpDir, opts.PolaPackage)
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("codegen: %w", err)
		}

		if codegenResult != nil && codegenResult.BridgePath != "" {
			replace[codegenResult.VirtualPath] = codegenResult.BridgePath
			tsOutPath = codegenResult.TSOutPath
		}

		if verbose && codegenResult != nil && codegenResult.TSOutPath != "" {
			fmt.Printf("Generated types: %s\n", codegenResult.TSOutPath)
		}
	} else if verbose {
		fmt.Println("No actions/ directory found, skipping codegen.")
	}

	// 2. Resolve the actions import path so pola_plugins.go can blank-import it.
	if hasActions {
		if modPath, err := readModulePath(projectDir); err == nil && modPath != "" {
			actionsImport = modPath + "/actions"
		}
	}

	// 3. Discover route packages under routes/.
	var routePkgs []routePackageInfo
	if modPath, err := readModulePath(projectDir); err == nil && modPath != "" {
		routePkgs = discoverRoutePackages(projectDir, modPath)
	}

	// 3b. Generate pola_route_init.go overlay for each route package.
	for i, rp := range routePkgs {
		src := generateRouteInit(rp.PkgName, opts.PolaPackage)
		initPath := filepath.Join(tmpDir, fmt.Sprintf("pola_route_init_%d.go", i))
		if err := os.WriteFile(initPath, src, 0o644); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("write route init for %s: %w", rp.PkgName, err)
		}
		replace[filepath.Join(rp.AbsDir, "pola_route_init.go")] = initPath
	}

	// 4. Generate plugin imports (always).
	pluginsSrc, err := generatePluginImports(opts, actionsImport, routePkgs)
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
func generatePluginImports(opts pluginOpts, actionsImport string, routePkgs []routePackageInfo) ([]byte, error) {
	routeImports := make([]string, len(routePkgs))
	for i, rp := range routePkgs {
		routeImports[i] = rp.ImportPath
	}

	hasCSS := opts.CSS != "" && opts.CSS != "none"
	hasCache := opts.Cache != "" && opts.Cache != "none"
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
		CSRF            bool
		SecurityHeaders bool
		Dev             bool
		Embed           bool
		HasRoutes       bool
		ActionsImport   string
		RouteImports    []string
	}{
		PolaPackage:     opts.PolaPackage,
		Engine:          opts.Engine,
		Bundler:         opts.Bundler,
		Renderer:        opts.Renderer,
		Router:          opts.Router,
		CSS:             condStr(hasCSS, opts.CSS, ""),
		Cache:           condStr(hasCache, opts.Cache, ""),
		CSRF:            hasCSRF,
		SecurityHeaders: hasSecurityHeaders,
		Dev:             opts.Dev,
		Embed:           opts.Embed,
		HasRoutes:       len(routePkgs) > 0,
		ActionsImport:   actionsImport,
		RouteImports:    routeImports,
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

func condStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// routePackageInfo holds metadata about a discovered route package.
type routePackageInfo struct {
	ImportPath string // e.g. "test-app/routes/kampala/uganda"
	AbsDir     string // e.g. "/abs/path/routes/kampala/uganda"
	PkgName    string // e.g. "uganda"
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
// registers the Route struct in the given package via init().
func generateRouteInit(pkgName, polaPackage string) []byte {
	return []byte(fmt.Sprintf(
		"// Code generated by pola; DO NOT EDIT.\npackage %s\n\nimport \"%s/routes\"\n\nfunc init() { routes.Register(&Route{}) }\n",
		pkgName, polaPackage,
	))
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

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/polagonow/pola/internal/autoload"
	"github.com/polagonow/pola/internal/autoload/pluginimports"
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/app"
	"github.com/polagonow/pola/internal/stubpkgs"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

// choiceToModule maps Polafile choice names to their Go module paths.
// Only choices backed by Go modules are listed here.
var choiceToModule = map[string]string{
	"goja":    "github.com/dop251/goja",
	"sobek":   "github.com/grafana/sobek",
	"quickjs": "github.com/buke/quickjs-go",
	"v8":      "rogchap.com/v8go",
	"esbuild": "github.com/evanw/esbuild",
}

// resolveVersion appends the Go module version to a choice name if available.
// For example, "goja" becomes "goja@v0.0.0-20260311135729-065cd970411c".
func resolveVersion(choice string) string {
	modPath, ok := choiceToModule[choice]
	if !ok {
		return choice
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return choice
	}
	for _, dep := range bi.Deps {
		if dep.Path == modPath {
			return polafile.FormatVersioned(choice, dep.Version)
		}
	}
	return choice
}

var newFlags struct {
	renderer        string
	bundler         string
	router          string
	css             string
	vm              string
	ui              string
	polaPath        string
	pm              string
	module          string
	testFramework   string
	skipTests       bool
	dependencies    string
	csrf            bool
	securityHeaders bool
	apiOnly         bool
}

var newCmd = &cobra.Command{
	Use:   "new [app-name]",
	Short: "Create a new Pola application",
	Long: `Scaffold a new Pola application with a working project structure,
including a Go server entry point, React app directory, and configuration files.

The app name can be a short identifier (e.g. "my-app") or a full Go module path
(e.g. "github.com/owner/my-app"). When a module path is given, the last segment
is used as the directory name and the full path becomes the Go module.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNew,
	Example: `  pola new                                          # prompts for name
  pola new my-app
  pola new github.com/acme/admin                    # name=admin, module=github.com/acme/admin
  pola new my-app --module github.com/acme/my-app
  pola new my-app --renderer=react --bundler=esbuild
  pola new my-app --css=tailwind
  pola new my-api --api-only                        # API-only, no frontend`,
}

func init() {
	newCmd.Flags().StringVar(&newFlags.renderer, "renderer", "react", "view renderer (react)")
	newCmd.Flags().StringVar(&newFlags.bundler, "bundler", "esbuild", "JS bundler (esbuild)")
	newCmd.Flags().StringVar(&newFlags.router, "router", "nextjs", "router style (nextjs)")
	newCmd.Flags().StringVar(&newFlags.css, "css", "none", "CSS processor (tailwind, sass, none)")
	newCmd.Flags().StringVar(&newFlags.vm, "vm", "goja", "JS engine (goja)")
	newCmd.Flags().StringVar(&newFlags.ui, "ui", "none", "UI component library (shadcn, mui, slds, ads, carbon, patternfly, fluentui, antd, none)")
	newCmd.Flags().StringVar(&newFlags.polaPath, "pola-path", "", "local path to pola framework source (adds replace directive)")
	newCmd.Flags().StringVar(&newFlags.pm, "pm", "", "package manager to use (npm, pnpm, yarn); auto-detected if not set")
	newCmd.Flags().StringVar(&newFlags.module, "module", "", "Go module path (e.g. github.com/owner/repo); auto-derived if the app name is a module path, otherwise defaults to the app name")
	newCmd.Flags().StringVar(&newFlags.dependencies, "dependencies", "", "comma-separated npm package version overrides (e.g. \"react@19.2.4,tailwindcss@^4.3.0\")")
	newCmd.Flags().BoolVar(&newFlags.csrf, "csrf", true, "enable CSRF protection")
	newCmd.Flags().BoolVar(&newFlags.securityHeaders, "security-headers", true, "enable security headers")
	newCmd.Flags().StringVar(&newFlags.testFramework, "test-framework", "vitest", "TS test framework (vitest, jest, none)")
	newCmd.Flags().BoolVar(&newFlags.skipTests, "skip-tests", false, "skip generating test files and test infrastructure")
	newCmd.Flags().BoolVar(&newFlags.apiOnly, "api-only", false, "create an API-only application (no frontend/web directory)")
}

func runNew(cmd *cobra.Command, args []string) error {
	var defaultInput string
	if len(args) == 1 {
		defaultInput = args[0]
	}

	var rawInput string
	prompt := &survey.Input{
		Message: "App name (or Go module path, e.g. github.com/owner/repo):",
		Default: defaultInput,
	}
	if err := survey.AskOne(prompt, &rawInput, survey.WithValidator(survey.Required)); err != nil {
		return fmt.Errorf("prompt: %w", err)
	}

	appName, parsedModule, err := parseAppNameAndModule(rawInput)
	if err != nil {
		return err
	}

	if newFlags.apiOnly {
		for _, flagName := range []string{"renderer", "bundler", "router", "css", "ui", "pm", "dependencies", "test-framework"} {
			if cmd.Flags().Lookup(flagName).Changed {
				return fmt.Errorf("--%s is not compatible with --api-only", flagName)
			}
		}
	}

	if !cmd.Flags().Lookup("ui").Changed {
		uiPrompt := &survey.Select{
			Message: "UI component library:",
			Options: []string{"none", "shadcn", "mui", "antd", "carbon", "patternfly", "fluentui", "slds", "ads"},
			Default: "none",
		}
		if err := survey.AskOne(uiPrompt, &newFlags.ui); err != nil {
			return fmt.Errorf("prompt ui: %w", err)
		}
	}

	modulePath := newFlags.module
	if modulePath == "" {
		modulePath = parsedModule
	}
	if modulePath == "" {
		modulePath = appName
	}

	targetDir, err := filepath.Abs(appName)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Check that the directory doesn't already exist.
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("directory %q already exists", appName)
	}

	if !newFlags.apiOnly {
		// Validate --ui flag (skip in API-only mode).
		if newFlags.ui == "shadcn" {
			if newFlags.css != "tailwind" {
				return fmt.Errorf("--ui=shadcn requires --css=tailwind")
			}
			if newFlags.renderer != "react" {
				return fmt.Errorf("--ui=shadcn requires --renderer=react")
			}
		}
		if newFlags.ui == "slds" {
			if newFlags.renderer != "react" {
				return fmt.Errorf("--ui=slds requires --renderer=react")
			}
		}
		if newFlags.ui == "ads" {
			if newFlags.renderer != "react" {
				return fmt.Errorf("--ui=ads requires --renderer=react")
			}
		}
		if newFlags.ui == "carbon" {
			if newFlags.renderer != "react" {
				return fmt.Errorf("--ui=carbon requires --renderer=react")
			}
		}
		if newFlags.ui == "fluentui" {
			if newFlags.renderer != "react" {
				return fmt.Errorf("--ui=fluentui requires --renderer=react")
			}
		}
		if newFlags.ui == "antd" {
			if newFlags.renderer != "react" {
				return fmt.Errorf("--ui=antd requires --renderer=react")
			}
		}

		// Auto-detect CSS requirement from the UI template's dependencies
		// (unless user explicitly set --css).
		if newFlags.ui != "" && newFlags.ui != "none" && !cmd.Flags().Lookup("css").Changed {
			if app.UIRequiresSass(newFlags.renderer, newFlags.ui) {
				newFlags.css = "sass"
			} else if app.UIRequiresTailwind(newFlags.renderer, newFlags.ui) {
				newFlags.css = "tailwind"
			}
		}

		// If the UI didn't dictate a CSS choice and the user didn't pass --css,
		// ask. Default is "none" so bare React stays Tailwind-free.
		if !cmd.Flags().Lookup("css").Changed &&
			!app.UIRequiresTailwind(newFlags.renderer, newFlags.ui) &&
			!app.UIRequiresSass(newFlags.renderer, newFlags.ui) {
			cssPrompt := &survey.Select{
				Message: "CSS framework:",
				Options: []string{"none", "tailwind"},
				Default: "none",
			}
			if err := survey.AskOne(cssPrompt, &newFlags.css); err != nil {
				return fmt.Errorf("prompt css: %w", err)
			}
		}
	}

	// Parse --dependencies before any filesystem work so user typos error out
	// before we create the target directory.
	npmOverrides, err := parseDependencies(newFlags.dependencies)
	if err != nil {
		return err
	}

	fmt.Printf("Creating %s...\n", appName)

	// If running from a dev build, detect the local pola source for a replace directive.
	polaLocalPath := findPolaSource()

	// Validate --test-framework.
	switch newFlags.testFramework {
	case "vitest", "jest", "none":
	default:
		return fmt.Errorf("invalid --test-framework %q: must be vitest, jest, or none", newFlags.testFramework)
	}

	generateTests := !newFlags.skipTests && newFlags.testFramework != "none"

	data := app.Data{
		AppName:       appName,
		ModulePath:    modulePath,
		PolaPackage:   polafile.DefaultPackage,
		Renderer:      newFlags.renderer,
		Bundler:       newFlags.bundler,
		Router:        newFlags.router,
		CSS:           newFlags.css,
		UI:            newFlags.ui,
		VM:            newFlags.vm,
		PolaVersion:   version,
		PolaLocalPath: polaLocalPath,
		GenerateTests: generateTests,
		TestFramework: newFlags.testFramework,
		NpmOverrides:  npmOverrides,
		APIOnly:       newFlags.apiOnly,
	}

	// Create the public directory (needed for asset embedding during builds).
	if !newFlags.apiOnly {
		if err := os.MkdirAll(filepath.Join(targetDir, "public"), 0o755); err != nil {
			return fmt.Errorf("create public dir: %w", err)
		}
	}

	// Execute scaffold templates.
	if err := app.Execute(targetDir, data); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	// Detect package manager early so we can persist it in the Polafile.
	pm := newFlags.pm
	if pm == "" {
		pm = detectPackageManager()
	}

	// Write Polafile.hcl to lock the user's choices with resolved versions.
	pf := &polafile.Polafile{
		Package:         modulePath,
		Version:         version,
		APIOnly:         newFlags.apiOnly,
		Routes:          "routes",
		Repositories:    "repositories",
		Services:        "services",
		CSRF:            &polafile.CSRF{Enabled: polafile.BoolPtr(newFlags.csrf)},
		SecurityHeaders: &polafile.SecurityHeaders{Enabled: polafile.BoolPtr(newFlags.securityHeaders)},
		Cache:           &polafile.Cache{Adapter: "memory"},
	}
	if !newFlags.apiOnly {
		pf.Actions = "actions"
		pf.Renderer = resolveVersion(newFlags.renderer)
		pf.Engine = resolveVersion(newFlags.vm)
		pf.Bundler = resolveVersion(newFlags.bundler)
		pf.Router = newFlags.router
		pf.CSS = newFlags.css
		pf.UI = newFlags.ui
		pf.PackageManager = pm
		pf.App = "web"
		pf.Testing = &polafile.Testing{
			GenerateTests: &generateTests,
			Framework:     newFlags.testFramework,
		}
	}
	if err := polafile.Save(targetDir, pf); err != nil {
		return fmt.Errorf("write Polafile.hcl: %w", err)
	}

	if !newFlags.apiOnly {
		// Write a placeholder favicon.
		faviconPath := filepath.Join(targetDir, "public", "favicon.ico")
		if _, err := os.Stat(faviconPath); os.IsNotExist(err) {
			_ = os.WriteFile(faviconPath, []byte{}, 0o644)
		}
	}

	// Write a temporary pola_plugins.go so go mod tidy resolves plugin deps.
	// This file is removed after tidy — at runtime it's injected via overlay.
	pluginOpts := autoload.PluginOpts{
		PolaPackage:     polafile.DefaultPackage,
		Cache:           "memory",
		CSRF:            newFlags.csrf,
		SecurityHeaders: newFlags.securityHeaders,
		Dev:             true,
		APIOnly:         newFlags.apiOnly,
	}
	if !newFlags.apiOnly {
		pluginOpts.Engine = newFlags.vm
		pluginOpts.Bundler = newFlags.bundler
		pluginOpts.Renderer = newFlags.renderer
		pluginOpts.Router = newFlags.router
		pluginOpts.CSS = newFlags.css
	}
	var actionsImport string
	if !newFlags.apiOnly {
		actionsImport = modulePath + "/actions"
	}
	pluginsPath := filepath.Join(targetDir, "pola_plugins.go")
	pluginsSrc, err := pluginimports.GenerateSource(pluginOpts, actionsImport, []string{
		modulePath + "/routes/health",
	}, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("generate plugins: %w", err)
	}
	if err := os.WriteFile(pluginsPath, pluginsSrc, 0o644); err != nil {
		return fmt.Errorf("write plugins: %w", err)
	}

	// Run go mod tidy.
	fmt.Println("Running go mod tidy...")
	if err := runInDir(targetDir, "go", "mod", "tidy"); err != nil {
		fmt.Printf("Warning: go mod tidy failed: %v\n", err)
		fmt.Println("You may need to run 'go mod tidy' manually.")
	}

	// Remove the temporary plugins file — overlay handles it at serve/build time.
	os.Remove(pluginsPath)

	if !newFlags.apiOnly {
		// Run package manager install in the web/ directory (frontend root).
		webDir := filepath.Join(targetDir, "web")
		fmt.Printf("Running %s install...\n", pm)
		if err := runInDir(webDir, pm, "install"); err != nil {
			fmt.Printf("Warning: %s install failed: %v\n", pm, err)
			fmt.Printf("You may need to run '%s install' manually.\n", pm)
		}

		// Stub @pola/actions and @pola/react into node_modules.
		if err := stubpkgs.StubToNodeModules(webDir); err != nil {
			fmt.Printf("Warning: failed to stub @pola packages: %v\n", err)
		}

		// Run js:bridge generator from the new project directory so it can
		// find the Polafile and go.mod.
		origDir, _ := os.Getwd()
		if err := os.Chdir(targetDir); err == nil {
			if err := generators.Run("js:bridge", nil, []string{}); err != nil {
				if verbose {
					fmt.Printf("js:bridge: %v\n", err)
				}
			}
			os.Chdir(origDir)
		}
	}

	// Print success message.
	fmt.Println()
	fmt.Printf("  %s is ready!\n\n", appName)
	fmt.Printf("  cd %s\n", appName)
	if newFlags.apiOnly {
		fmt.Println("  pola serve        # start API server")
	} else {
		fmt.Println("  pola serve        # start dev server")
	}
	fmt.Println("  pola build        # build for production")
	fmt.Println()

	return nil
}

// parseDependencies parses a comma-separated list of "name@version" entries
// (e.g. "react@19.2.4,@types/react@^19.0.0") into a map. Returns an empty
// map for empty input. Errors on malformed entries or unknown package names.
func parseDependencies(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	out := make(map[string]string)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, ver, ok := splitNameVersion(entry)
		if !ok {
			return nil, fmt.Errorf("invalid --dependencies entry %q: expected name@version", entry)
		}
		if !app.IsKnownNpmPackage(name) {
			return nil, fmt.Errorf("unknown dependency %q; valid names: %s",
				name, strings.Join(app.KnownNpmPackages(), ", "))
		}
		out[name] = ver
	}
	return out, nil
}

// splitNameVersion splits "name@version" into its parts, correctly handling
// scoped packages like "@types/react@^19.0.0" (where the leading "@" is part
// of the name). Returns ok=false if no version separator is found.
func splitNameVersion(s string) (name, version string, ok bool) {
	start := 0
	if strings.HasPrefix(s, "@") {
		start = 1
	}
	i := strings.IndexByte(s[start:], '@')
	if i < 0 {
		return "", "", false
	}
	sep := start + i
	name = s[:sep]
	version = s[sep+1:]
	if name == "" || version == "" {
		return "", "", false
	}
	return name, version, true
}

// parseAppNameAndModule splits a raw user input into a short app name and a Go
// module path. If the input contains "/" it's treated as a module path: the
// last non-empty path segment becomes the app name and the full cleaned input
// becomes the module path. Otherwise the input is used as-is and modulePath is
// left empty for the caller to fill in. Strips leading "http://"/"https://"
// and a trailing ".git".
func parseAppNameAndModule(raw string) (appName, modulePath string, err error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "https://")
	cleaned = strings.TrimPrefix(cleaned, "http://")
	cleaned = strings.TrimSuffix(cleaned, ".git")
	cleaned = strings.TrimRight(cleaned, "/")

	if cleaned == "" {
		return "", "", fmt.Errorf("app name is required")
	}

	if strings.Contains(cleaned, "/") {
		idx := strings.LastIndex(cleaned, "/")
		name := cleaned[idx+1:]
		if name == "" {
			return "", "", fmt.Errorf("invalid module path %q: empty last segment", raw)
		}
		return name, cleaned, nil
	}

	return cleaned, "", nil
}

// detectPackageManager returns the best available JS package manager.
func detectPackageManager() string {
	for _, pm := range []string{"pnpm", "yarn", "npm"} {
		if _, err := exec.LookPath(pm); err == nil {
			return pm
		}
	}
	return "npm"
}

// runInDir executes a command in the given directory.
func runInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// findPolaSource returns the absolute path to the local pola framework source,
// or empty string if not found. Checks --pola-path flag first, then walks up
// from the executable looking for the pola module.
func findPolaSource() string {
	if newFlags.polaPath != "" {
		abs, err := filepath.Abs(newFlags.polaPath)
		if err != nil {
			return newFlags.polaPath
		}
		return abs
	}

	// If this is a dev build, try to find the pola source by looking for
	// go.mod with module github.com/polagonow/pola near the executable.
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 5; i++ {
		gomod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(gomod); err == nil {
			content := string(data)
			for _, line := range strings.Split(content, "\n") {
				if strings.TrimSpace(line) == "module github.com/polagonow/pola" {
					return dir
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

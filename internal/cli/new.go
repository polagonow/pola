package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/internal/cli/scaffold"
	"github.com/polagonow/pola/internal/cli/stubpkgs"
	"github.com/spf13/cobra"
)

var newFlags struct {
	renderer string
	bundler  string
	router   string
	css      string
	vm       string
	polaPath string
	pm       string
}

var newCmd = &cobra.Command{
	Use:   "new <app-name>",
	Short: "Create a new Pola application",
	Long: `Scaffold a new Pola application with a working project structure,
including a Go server entry point, React app directory, and configuration files.`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
	Example: `  pola new my-app
  pola new my-app --renderer=react --bundler=esbuild
  pola new my-app --css=tailwind`,
}

func init() {
	newCmd.Flags().StringVar(&newFlags.renderer, "renderer", "react", "view renderer (react)")
	newCmd.Flags().StringVar(&newFlags.bundler, "bundler", "esbuild", "JS bundler (esbuild)")
	newCmd.Flags().StringVar(&newFlags.router, "router", "nextjs", "router style (nextjs)")
	newCmd.Flags().StringVar(&newFlags.css, "css", "tailwind", "CSS processor (tailwind, none)")
	newCmd.Flags().StringVar(&newFlags.vm, "vm", "goja", "JS engine (goja)")
	newCmd.Flags().StringVar(&newFlags.polaPath, "pola-path", "", "local path to pola framework source (adds replace directive)")
	newCmd.Flags().StringVar(&newFlags.pm, "pm", "", "package manager to use (npm, pnpm, yarn); auto-detected if not set")
}

func runNew(_ *cobra.Command, args []string) error {
	appName := args[0]
	targetDir, err := filepath.Abs(appName)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Check that the directory doesn't already exist.
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("directory %q already exists", appName)
	}

	fmt.Printf("Creating %s...\n", appName)

	// If running from a dev build, detect the local pola source for a replace directive.
	polaLocalPath := findPolaSource()

	data := scaffold.Data{
		AppName:       appName,
		ModulePath:    appName,
		Renderer:      newFlags.renderer,
		Bundler:       newFlags.bundler,
		Router:        newFlags.router,
		CSS:           newFlags.css,
		VM:            newFlags.vm,
		PolaVersion:   version,
		PolaLocalPath: polaLocalPath,
	}

	// Create the public directory (needed for embed.go).
	if err := os.MkdirAll(filepath.Join(targetDir, "public"), 0o755); err != nil {
		return fmt.Errorf("create public dir: %w", err)
	}

	// Execute scaffold templates.
	if err := scaffold.Execute(targetDir, data); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	// Write a placeholder favicon.
	faviconPath := filepath.Join(targetDir, "public", "favicon.ico")
	if _, err := os.Stat(faviconPath); os.IsNotExist(err) {
		_ = os.WriteFile(faviconPath, []byte{}, 0o644)
	}

	// Write a temporary pola_plugins.go so go mod tidy resolves plugin deps.
	// This file is removed after tidy — at runtime it's injected via overlay.
	pluginsPath := filepath.Join(targetDir, "pola_plugins.go")
	pluginsSrc := generatePluginImports(newFlags.css)
	if err := os.WriteFile(pluginsPath, pluginsSrc, 0o644); err != nil {
		fmt.Printf("Warning: failed to write temp plugins file: %v\n", err)
	}

	// Run go mod tidy.
	fmt.Println("Running go mod tidy...")
	if err := runInDir(targetDir, "go", "mod", "tidy"); err != nil {
		fmt.Printf("Warning: go mod tidy failed: %v\n", err)
		fmt.Println("You may need to run 'go mod tidy' manually.")
	}

	// Remove the temporary plugins file — overlay handles it at serve/build time.
	os.Remove(pluginsPath)

	// Detect and run package manager install.
	pm := newFlags.pm
	if pm == "" {
		pm = detectPackageManager()
	}
	fmt.Printf("Running %s install...\n", pm)
	if err := runInDir(targetDir, pm, "install"); err != nil {
		fmt.Printf("Warning: %s install failed: %v\n", pm, err)
		fmt.Printf("You may need to run '%s install' manually.\n", pm)
	}

	// Stub @pola/di and @pola/react into node_modules.
	if err := stubpkgs.StubToNodeModules(targetDir); err != nil {
		fmt.Printf("Warning: failed to stub @pola packages: %v\n", err)
	}

	// Print success message.
	fmt.Println()
	fmt.Printf("  %s is ready!\n\n", appName)
	fmt.Printf("  cd %s\n", appName)
	fmt.Println("  pola serve        # start dev server")
	fmt.Println("  pola build        # build for production")
	fmt.Println()

	return nil
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

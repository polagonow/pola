package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/polagonow/pola/internal/cli/buildtags"
	"github.com/spf13/cobra"
)

var serveFlags struct {
	port     string
	renderer string
	bundler  string
	router   string
	css      string
	vm       string
	appPath  string
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the development server",
	Long:  "Run the Pola app in development mode with hot reload.",
	RunE:  runServe,
	Example: `  pola serve
  pola serve --port 8080
  pola serve --vm goja --css tailwind`,
	Aliases: []string{"dev"},
}

func init() {
	serveCmd.Flags().StringVarP(&serveFlags.port, "port", "p", envOr("PORT", "3000"), "server port")
	serveCmd.Flags().StringVar(&serveFlags.renderer, "renderer", envOr("POLA_RENDERER", "react"), "view renderer")
	serveCmd.Flags().StringVar(&serveFlags.bundler, "bundler", envOr("POLA_BUNDLER", "esbuild"), "JS bundler")
	serveCmd.Flags().StringVar(&serveFlags.router, "router", envOr("POLA_ROUTER", "nextjs"), "router style")
	serveCmd.Flags().StringVar(&serveFlags.css, "css", envOr("POLA_CSS", "tailwind"), "CSS processor")
	serveCmd.Flags().StringVar(&serveFlags.vm, "vm", envOr("POLA_VM", "goja"), "JS engine")
	serveCmd.Flags().StringVar(&serveFlags.appPath, "app-path", envOr("POLA_WEBAPP_PATH", "./app"), "path to the web app directory")
}

func runServe(_ *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Project root: %s\n", projectDir)
	}

	// Run action bridge codegen if actions/ directory exists.
	if err := runCodegen(projectDir); err != nil {
		return err
	}

	// Run templ generate if templ files exist.
	if hasTemplFiles(projectDir) {
		fmt.Println("Generating templ components...")
		if err := runInDir(projectDir, "go", "tool", "templ", "generate", "./shell/..."); err != nil {
			if verbose {
				fmt.Printf("Warning: templ generate failed: %v\n", err)
			}
		}
	}

	tags := buildtags.RuntimeTags(serveFlags.vm, serveFlags.bundler, serveFlags.renderer, serveFlags.router, serveFlags.css)
	if verbose {
		fmt.Printf("Build tags: %s\n", tags)
	}

	fmt.Printf("Starting dev server on port %s...\n", serveFlags.port)

	// Build the go run command.
	goArgs := []string{"run", "-tags", tags, "."}
	cmd := exec.Command("go", goArgs...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"POLA_DEV=true",
		"PORT="+serveFlags.port,
		"POLA_WEBAPP_PATH="+serveFlags.appPath,
	)

	// Start the process.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	// Forward signals to child process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case sig := <-sigCh:
		_ = cmd.Process.Signal(sig)
		return <-doneCh
	case err := <-doneCh:
		return err
	}
}

// findProjectRoot walks up from cwd looking for a go.mod file.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fall back to cwd.
	return os.Getwd()
}

// hasTemplFiles checks if there are any .templ files in the project.
func hasTemplFiles(dir string) bool {
	shellDir := filepath.Join(dir, "shell")
	if _, err := os.Stat(shellDir); os.IsNotExist(err) {
		return false
	}
	entries, err := os.ReadDir(shellDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".templ" {
			return true
		}
	}
	return false
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

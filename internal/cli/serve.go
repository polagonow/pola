package cli

import (
	"cmp"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/polagonow/pola/internal/autoload"
	_ "github.com/polagonow/pola/internal/autoload/all" // register all autoloads
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/internal/stubpkgs"
	"github.com/polagonow/pola/polafile"
	"github.com/polagonow/pola/watcher"
	"github.com/spf13/cobra"
)

var serveFlags struct {
	port            string
	renderer        string
	bundler         string
	router          string
	css             string
	vm              string
	appPath         string
	csrf            bool
	securityHeaders bool
	imageProcessing string
}

var serveCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server",
	Long:  "Run the Pola app in development mode with hot reload.",
	RunE:  runServe,
	Example: `  pola dev
  pola dev --port 8080
  pola dev --vm goja --css tailwind`,
	Aliases: []string{"serve"},
}

func init() {
	serveCmd.Flags().StringVarP(&serveFlags.port, "port", "p", cmp.Or(os.Getenv("PORT"), "3000"), "server port")
	serveCmd.Flags().StringVar(&serveFlags.renderer, "renderer", cmp.Or(os.Getenv("POLA_RENDERER"), "react"), "view renderer")
	serveCmd.Flags().StringVar(&serveFlags.bundler, "bundler", cmp.Or(os.Getenv("POLA_BUNDLER"), "esbuild"), "JS bundler")
	serveCmd.Flags().StringVar(&serveFlags.router, "router", cmp.Or(os.Getenv("POLA_ROUTER"), "nextjs"), "router style")
	serveCmd.Flags().StringVar(&serveFlags.css, "css", cmp.Or(os.Getenv("POLA_CSS"), "tailwind"), "CSS processor")
	serveCmd.Flags().StringVar(&serveFlags.vm, "vm", cmp.Or(os.Getenv("POLA_VM"), "goja"), "JS engine")
	serveCmd.Flags().StringVar(&serveFlags.appPath, "app-path", cmp.Or(os.Getenv("POLA_WEBAPP_PATH"), "./web"), "path to the web app directory")
	serveCmd.Flags().BoolVar(&serveFlags.csrf, "csrf", os.Getenv("POLA_CSRF") != "false", "enable CSRF protection")
	serveCmd.Flags().BoolVar(&serveFlags.securityHeaders, "security-headers", os.Getenv("POLA_SECURITY_HEADERS") != "false", "enable security headers")
	serveCmd.Flags().StringVar(&serveFlags.imageProcessing, "image-processing", os.Getenv("POLA_IMAGE_PROCESSING"), "image processing adapter")
}

// goWatchExts are the file extensions that trigger a Go process restart.
var goWatchExts = []string{".go", ".tmpl"}

func runServe(cmd *cobra.Command, _ []string) error {
	projectDir, err := findProjectRoot()
	if err != nil {
		return err
	}

	applyPolafileDefaults(cmd, projectDir)

	// Load Polafile for PolaPackage.
	var pf polafile.Polafile
	pfPtr, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pfPtr != nil {
		pf = *pfPtr
	}

	isAPIOnly := pfPtr != nil && pfPtr.IsAPIOnly()

	if verbose {
		fmt.Printf("Project root: %s\n", projectDir)
	}

	if !isAPIOnly {
		// Auto-install frontend dependencies if node_modules is missing.
		webDir := filepath.Join(projectDir, serveFlags.appPath)
		if _, err := os.Stat(filepath.Join(webDir, "node_modules")); os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(webDir, "package.json")); err == nil {
				pm := pf.PackageManager
				if pm == "" {
					pm = detectPackageManager()
				}
				if i := strings.IndexByte(pm, '@'); i > 0 {
					pm = pm[:i]
				}
				fmt.Printf("Installing frontend dependencies (%s install)...\n", pm)
				if err := runInDir(webDir, pm, "install"); err != nil {
					return fmt.Errorf("%s install: %w", pm, err)
				}
			}
		}

		// Stub @pola/actions and @pola/react into node_modules.
		if err := stubpkgs.StubToNodeModules(filepath.Join(projectDir, serveFlags.appPath)); err != nil {
			return fmt.Errorf("stub packages: %w", err)
		}

		// Run js:bridge generator to produce TypeScript declarations.
		if err := generators.Run("js:bridge", nil, []string{}); err != nil {
			if verbose {
				fmt.Printf("js:bridge: %v\n", err)
			}
		}
	}

	printStartupBanner(projectDir, serveFlags.port)

	// Forward OS signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Restart loop: on .go/.tmpl file change → kill → regenerate overlay → respawn.
	for {
		// Generate overlay (plugin imports + action bridge codegen).
		opts := autoload.PluginOpts{
			PolaPackage:     pf.PolaPackage(),
			Cache:           "memory",
			CSRF:            serveFlags.csrf,
			SecurityHeaders: serveFlags.securityHeaders,
			ImageProcessing: serveFlags.imageProcessing,
			Dev:             true,
			ActionsDir:      generateFlags.actionsDir,
			TSOut:           generateFlags.tsOut,
			APIOnly:         isAPIOnly,
		}
		if !isAPIOnly {
			opts.Engine = serveFlags.vm
			opts.Bundler = serveFlags.bundler
			opts.Renderer = serveFlags.renderer
			opts.Router = serveFlags.router
			opts.CSS = serveFlags.css
			opts.AppDir = pf.AppDir()
		}

		defaultEnv := "development"
		autoload.PopulateDatabaseOpts(&opts, &pf, defaultEnv)
		autoload.PopulateStorageOpts(&opts, &pf, defaultEnv)
		autoload.ApplyMailerOpts(&opts, &pf, defaultEnv)
		autoload.PopulateMCPOpts(&opts, &pf, defaultEnv)
		autoload.PopulateSessionOpts(&opts, &pf, defaultEnv)
		autoload.PopulateRateLimitOpts(&opts, &pf, defaultEnv)
		autoload.PopulateFlashOpts(&opts, &pf, defaultEnv)
		autoload.PopulateI18nOpts(&opts, &pf, defaultEnv)
		overlayRes, err := autoload.Run(projectDir, opts, verbose)
		if err != nil {
			return err
		}

		cmd := buildGoRunCmd(projectDir, overlayRes)
		if err := cmd.Start(); err != nil {
			cleanupOverlay(overlayRes)
			return fmt.Errorf("start server: %w", err)
		}

		// Start polling .go and .tmpl files for changes.
		restartCh := make(chan struct{}, 1)
		poller := watcher.NewWithCollect(func() []string {
			return collectGoFiles(projectDir)
		}, func() {
			select {
			case restartCh <- struct{}{}:
			default:
			}
		})
		poller.Start()

		doneCh := make(chan error, 1)
		go func() {
			doneCh <- cmd.Wait()
		}()

		select {
		case sig := <-sigCh:
			// User pressed Ctrl+C — forward signal to process group and exit.
			poller.Stop()
			killProcessGroup(cmd)
			cleanupOverlay(overlayRes)
			_ = sig
			return <-doneCh

		case err := <-doneCh:
			// Process exited on its own (crash, compile error, etc.).
			poller.Stop()
			cleanupOverlay(overlayRes)
			return err

		case <-restartCh:
			// .go or .tmpl file changed — kill process group, cleanup, restart.
			poller.Stop()
			killProcessGroup(cmd)
			<-doneCh
			cleanupOverlay(overlayRes)
			fmt.Print("\n  \033[36m↻ Go files changed, restarting...\033[0m\n\n")
			// Brief pause to let the OS release the listen port.
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// buildGoRunCmd creates the exec.Cmd for `go run` with overlay.
func buildGoRunCmd(projectDir string, overlayRes *autoload.Result) *exec.Cmd {
	goArgs := []string{"run"}
	if overlayRes != nil && overlayRes.OverlayPath != "" {
		goArgs = append(goArgs, "-overlay", overlayRes.OverlayPath)
	}
	goArgs = append(goArgs, ".")

	cmd := exec.Command("go", goArgs...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcessGroup(cmd)
	cmd.Env = append(os.Environ(),
		"POLA_ENV=development",
		"PORT="+serveFlags.port,
		"POLA_WEBAPP_PATH="+serveFlags.appPath,
		"GONOSUMCHECK=*",
		"GONOSUMDB=*",
		"GOFLAGS=-mod=mod",
	)
	return cmd
}

// cleanupOverlay removes the temporary overlay directory.
func cleanupOverlay(res *autoload.Result) {
	if res != nil && res.TmpDir != "" {
		os.RemoveAll(res.TmpDir)
	}
}

// collectGoFiles returns .go and .tmpl file paths in projectDir, plus go.mod
// and go.sum for dependency change detection.
func collectGoFiles(projectDir string) []string {
	paths := watcher.CollectPaths(projectDir, goWatchExts)
	// Also watch go.mod, go.sum, and Polafile.hcl for changes.
	for _, name := range []string{"go.mod", "go.sum", "Polafile.hcl"} {
		p := filepath.Join(projectDir, name)
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
	}
	return paths
}

// findProjectRoot is a convenience wrapper around project.FindRoot.
func findProjectRoot() (string, error) {
	return project.FindRoot()
}

// printStartupBanner displays a Next.js-style startup banner.
func printStartupBanner(projectDir, port string) {
	fmt.Printf("\n  \033[1mPola %s\033[0m\n\n", version)
	fmt.Printf("  - Local:        http://localhost:%s\n", port)

	if ip := outboundIP(); ip != "" {
		fmt.Printf("  - Network:      http://%s:%s\n", ip, port)
	}

	if envFiles := detectEnvFiles(projectDir); len(envFiles) > 0 {
		fmt.Printf("  - Environments: %s\n", strings.Join(envFiles, ", "))
	}

	fmt.Println()
}

// outboundIP returns the first non-loopback IPv4 address, or "" if none found.
func outboundIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}
	return ""
}

// detectEnvFiles checks for common .env files in the project directory.
func detectEnvFiles(projectDir string) []string {
	candidates := []string{
		".env",
		".env.local",
		".env.development",
		".env.development.local",
	}
	var found []string
	for _, name := range candidates {
		if _, err := os.Stat(filepath.Join(projectDir, name)); err == nil {
			found = append(found, name)
		}
	}
	return found
}

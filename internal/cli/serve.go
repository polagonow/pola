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

	"github.com/polagonow/pola/internal/stubpkgs"
	"github.com/polagonow/pola/internal/project"
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
	serveCmd.Flags().StringVar(&serveFlags.appPath, "app-path", cmp.Or(os.Getenv("POLA_WEBAPP_PATH"), "./app"), "path to the web app directory")
	serveCmd.Flags().BoolVar(&serveFlags.csrf, "csrf", os.Getenv("POLA_CSRF") != "false", "enable CSRF protection")
	serveCmd.Flags().BoolVar(&serveFlags.securityHeaders, "security-headers", os.Getenv("POLA_SECURITY_HEADERS") != "false", "enable security headers")
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
	if loaded, err := polafile.Load(projectDir); err == nil && loaded != nil {
		pf = *loaded
	}

	if verbose {
		fmt.Printf("Project root: %s\n", projectDir)
	}

	// Stub @pola/actions and @pola/react into node_modules.
	if err := stubpkgs.StubToNodeModules(projectDir); err != nil {
		return fmt.Errorf("stub packages: %w", err)
	}

	printStartupBanner(projectDir, serveFlags.port)

	// Forward OS signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Restart loop: on .go/.tmpl file change → kill → regenerate overlay → respawn.
	for {
		// Generate overlay (plugin imports + action bridge codegen).
		opts := pluginOpts{
			PolaPackage:     pf.PolaPackage(),
			Engine:          serveFlags.vm,
			Bundler:         serveFlags.bundler,
			Renderer:        serveFlags.renderer,
			Router:          serveFlags.router,
			CSS:             serveFlags.css,
			Cache:           "memory",
			CSRF:            serveFlags.csrf,
			SecurityHeaders: serveFlags.securityHeaders,
			Dev:             true,
		}
		populateDatabaseOpts(&opts, &pf, "development")
		overlayRes, err := generateOverlay(projectDir, opts)
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
func buildGoRunCmd(projectDir string, overlayRes *overlayResult) *exec.Cmd {
	goArgs := []string{"run"}
	if overlayRes != nil && overlayRes.OverlayPath != "" {
		goArgs = append(goArgs, "-overlay", overlayRes.OverlayPath)
	}
	goArgs = append(goArgs, ".")

	cmd := exec.Command("go", goArgs...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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

// killProcessGroup sends SIGTERM to the entire process group (the go run
// process and any child it spawned). Uses negative PID to target the group.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

// cleanupOverlay removes the temporary overlay directory.
func cleanupOverlay(res *overlayResult) {
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

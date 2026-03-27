package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/polagonow/pola/core"
)

// prebuildMeta is the JSON schema for prebuild-meta.json written during mage Bundle
// and read back at startup in embed mode.
type prebuildMeta struct {
	Routes         []core.Route      `json:"routes"`
	ClientEntryURL string            `json:"clientEntryURL"`
	ImportURLs     map[string]string `json:"importURLs"`
	GlobalNotFound string            `json:"globalNotFound"`
	CSSURL         string            `json:"cssURL,omitempty"`
}

// entryGenerator is the optional interface a Renderer may implement to produce
// the server-entry TypeScript source from a DiscoveryResult.
type entryGenerator interface {
	GenerateEntry(core.DiscoveryResult) (string, error)
}

// bundleConfigurator is the optional interface a Renderer may implement to
// provide extra BundleInput fields: esbuild conditions for the server pass and
// the client-side hydration entry point.
type bundleConfigurator interface {
	BundleConditions() []string
	ClientEntry() string
}

// discoveryProvider is the optional interface a Router may implement to expose
// the DiscoveryResult produced during ScanRoutes.
type discoveryProvider interface {
	DiscoveryResult() core.DiscoveryResult
}


// noopAssetServer is used when no asset plugin is registered.
type noopAssetServer struct{}

func (noopAssetServer) Handler(_ string) http.Handler { return http.NotFoundHandler() }

// noopShell is used when no HTML shell is registered.
type noopShell struct{}

func (noopShell) Render(_ core.ShellParams) string { return "<!DOCTYPE html><html><body></body></html>" }

// Build runs the full build pipeline and returns a ready App.
func Build(cfg *core.Config) (*core.App, error) {
	// ── 0. Resolve registry ───────────────────────────────────────────────
	reg := cfg.Registry
	if reg == nil {
		reg = &core.Registry{}
	}

	// ── 1. Fill defaults ──────────────────────────────────────────────────
	if err := reg.FillDefaults(); err != nil {
		return nil, fmt.Errorf("pola: fill defaults: %w", err)
	}

	// Propagate logger to all components that accept it.
	for _, c := range []any{reg.Engine, reg.Bundler, reg.Renderer, reg.Router, reg.CSS} {
		if la, ok := c.(core.LogAware); ok {
			la.SetLogger(reg.Logger)
		}
	}

	// ── Prebuild fast-path (embed mode) ───────────────────────────────────
	// When a prebuild loader is registered (via cmd/server/embed.go), skip
	// route scanning and JS bundling — everything was pre-computed at mage Build time.
	if loader := core.DefaultPrebuildLoader(); loader != nil {
		return buildFromPrebuilt(cfg, reg, loader)
	}

	// Validate required components.
	if reg.Router == nil {
		return nil, fmt.Errorf("pola: no Router registered; import github.com/polagonow/pola/router/nextjs (or another Router)")
	}
	if reg.Bundler == nil {
		return nil, fmt.Errorf("pola: no Bundler registered; import github.com/polagonow/pola/bundler/esbuild (or another Bundler)")
	}
	if reg.Renderer == nil {
		return nil, fmt.Errorf("pola: no Renderer registered; import github.com/polagonow/pola/renderer/react (or another Renderer)")
	}
	if reg.FS == nil {
		return nil, fmt.Errorf("pola: no FS registered; import github.com/polagonow/pola/fs/osfs (or another FS)")
	}

	// ── Resolve paths ─────────────────────────────────────────────────────
	webAppPath := cfg.WebAppPath
	if webAppPath == "" {
		webAppPath = "./app"
	}
	absWebAppPath, err := filepath.Abs(webAppPath)
	if err != nil {
		return nil, fmt.Errorf("pola: resolve web app path: %w", err)
	}

	publicDir := cfg.PublicDir
	if publicDir == "" {
		publicDir = filepath.Join(absWebAppPath, "public")
	}
	assetsURLPath := "/public/assets"

	ctx := context.Background()

	// ── 2. Scan routes ────────────────────────────────────────────────────
	exts := reg.Renderer.FileExtensions()
	routes, err := reg.Router.ScanRoutes(ctx, reg.FS, absWebAppPath, exts)
	if err != nil {
		return nil, fmt.Errorf("pola: scan routes: %w", err)
	}

	// ── 3. Get DiscoveryResult (optional, router may implement it) ────────
	var discovery core.DiscoveryResult
	if dp, ok := reg.Router.(discoveryProvider); ok {
		discovery = dp.DiscoveryResult()
	}

	// ── 4. Generate server entry source ───────────────────────────────────
	var serverEntryContent string
	if eg, ok := reg.Renderer.(entryGenerator); ok {
		serverEntryContent, err = eg.GenerateEntry(discovery)
		if err != nil {
			return nil, fmt.Errorf("pola: generate entry: %w", err)
		}
	}

	// ── 5. Collect public env vars ─────────────────────────────────────────
	publicEnvVars := make(map[string]string)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "POLA_PUBLIC_") {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				publicEnvVars[parts[0]] = parts[1]
			}
		}
	}

	// ── 6. Bundle ──────────────────────────────────────────────────────────
	bundleInput := core.BundleInput{
		AppDir:             absWebAppPath,
		OutDir:             filepath.Join(publicDir, "assets"),
		AssetsURLPath:      assetsURLPath,
		ClientComponents:   discovery.ClientComponents,
		Dev:                cfg.Dev,
		ServerEntryContent: serverEntryContent,
		PublicEnvVars:      publicEnvVars,
	}
	if bc, ok := reg.Renderer.(bundleConfigurator); ok {
		bundleInput.ServerBundleConditions = bc.BundleConditions()
		bundleInput.ClientEntry = bc.ClientEntry()
	}

	bundleOutput, err := reg.Bundler.Build(ctx, bundleInput)
	if err != nil {
		return nil, fmt.Errorf("pola: bundle: %w", err)
	}

	// ── 6.5. Process CSS ─────────────────────────────────────────────────
	// Scan source files for CSS imports (like Next.js: `import "./styles.css"`),
	// generate a synthetic input that @import's them all, and pass it to the
	// CSS processor (e.g. Tailwind).
	var cssURL string
	if reg.CSS != nil {
		cssFiles := scanCSSImports(reg.FS, absWebAppPath, exts)
		if len(cssFiles) > 0 {
			cssInput, cleanup, err := generateCSSInput(cssFiles, absWebAppPath)
			if err != nil {
				return nil, fmt.Errorf("pola: css input: %w", err)
			}
			defer cleanup()

			cssOutput := filepath.Join(publicDir, "assets", "styles.css")
			if err := os.MkdirAll(filepath.Dir(cssOutput), 0o755); err != nil {
				return nil, fmt.Errorf("pola: css mkdir: %w", err)
			}
			if err := reg.CSS.Process(ctx, cssInput, cssOutput); err != nil {
				return nil, fmt.Errorf("pola: css: %w", err)
			}
			cssURL = assetsURLPath + "/styles.css"
		}
	}

	// ── 6.6. Wire bundle to renderer ──────────────────────────────────────
	// Renderers that implement core.BundleLoader receive the compiled server
	// bundle so they can initialise their JS runtime pool.
	if bl, ok := reg.Renderer.(core.BundleLoader); ok && reg.Engine != nil && bundleOutput != nil {
		if err := bl.LoadBundle(reg.Engine, bundleOutput.ServerBundle); err != nil {
			return nil, fmt.Errorf("pola: load bundle: %w", err)
		}
	}

	// ── 7. Resolve shell and asset server ─────────────────────────────────
	var shell core.HTMLShell
	if ctor := core.DefaultHTMLShell(); ctor != nil {
		shell = ctor()
	} else {
		shell = noopShell{}
	}

	var assets core.AssetServer
	if ctor := core.DefaultAssetServer(); ctor != nil {
		assets = ctor(publicDir)
	} else {
		assets = noopAssetServer{}
	}

	// ── 8. Wire orchestrator ───────────────────────────────────────────────
	var notFoundRoute *core.Route
	if discovery.GlobalNotFound != "" {
		notFoundRoute = &core.Route{Export: "GlobalNotFound", Pattern: "/*"}
	}
	orch := NewOrchestrator(reg, routes, shell, assets, bundleOutput, notFoundRoute, cssURL, cfg.Dev)

	// ── 6.6. Copy static public files ────────────────────────────────────────
	// When publicDir differs from the webapp's own public/ folder (e.g. embed
	// builds writing to cmd/server/public/), copy non-asset statics such as
	// favicon.ico so the //go:embed picks them up.
	srcPublic := filepath.Join(absWebAppPath, "public")
	if srcPublic != publicDir {
		copyPublicStatics(srcPublic, publicDir)
	}

	// ── 9. Write prebuild-meta.json for embed builds ───────────────────────
	// Saved into publicDir so it's picked up by //go:embed public in cmd/server/embed.go.
	if bundleOutput != nil {
		meta := prebuildMeta{
			Routes:         routes,
			ClientEntryURL: bundleOutput.ClientEntryURL,
			ImportURLs:     bundleOutput.ImportURLs,
			GlobalNotFound: discovery.GlobalNotFound,
			CSSURL:         cssURL,
		}
		if b, err := json.Marshal(meta); err == nil {
			_ = os.WriteFile(filepath.Join(publicDir, "prebuild-meta.json"), b, 0o644)
		}
	}

	// ── 10. Return App ─────────────────────────────────────────────────────
	app := newApp(cfg, reg, orch)
	app.SetArtifacts(bundleOutput)

	// In dev mode wrap the app in a hot-reloader so file changes trigger a
	// full rebuild and browser reload via WebSocket.
	if cfg.Dev {
		hr, err := NewHotReloader(cfg, app)
		if err != nil {
			return nil, fmt.Errorf("pola: hotreload: %w", err)
		}
		devApp := newApp(cfg, reg, hr.Handler())
		devApp.SetArtifacts(bundleOutput)
		return devApp, nil
	}

	return app, nil
}

// New creates and builds a Pola application from the given config.
func New(cfg *core.Config) (*core.App, error) {
	return Build(cfg)
}

// copyPublicStatics copies non-asset static files from srcDir to dstDir,
// skipping the "assets" subdirectory which is managed by the bundler.
func copyPublicStatics(srcDir, dstDir string) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return
	}
	for _, e := range entries {
		if e.Name() == "assets" {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())
		if e.IsDir() {
			copyPublicStatics(src, dst)
		} else {
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			_ = os.WriteFile(dst, data, 0o644)
		}
	}
}

// routeLoader is the optional interface a Router may implement to accept
// pre-built routes without calling ScanRoutes (used in embed/prebuild mode).
type routeLoader interface {
	LoadRoutes([]core.Route)
}

// buildFromPrebuilt builds an App from pre-computed artifacts (embed mode).
// It skips route scanning, entry generation, and JS bundling.
func buildFromPrebuilt(cfg *core.Config, reg *core.Registry, loader func() (*core.PrebuildArtifacts, error)) (*core.App, error) {
	artifacts, err := loader()
	if err != nil {
		return nil, fmt.Errorf("pola: load prebuilt: %w", err)
	}

	// Seed the router's internal routing table from the pre-built route list.
	if rl, ok := reg.Router.(routeLoader); ok {
		rl.LoadRoutes(artifacts.Routes)
	}

	// Load the server bundle into the JS engine.
	if bl, ok := reg.Renderer.(core.BundleLoader); ok && reg.Engine != nil && artifacts.BundleOutput != nil {
		if err := bl.LoadBundle(reg.Engine, artifacts.BundleOutput.ServerBundle); err != nil {
			return nil, fmt.Errorf("pola: load bundle: %w", err)
		}
	}

	// Resolve shell and asset server (embed.go registers the embedded asset server).
	var shell core.HTMLShell
	if ctor := core.DefaultHTMLShell(); ctor != nil {
		shell = ctor()
	} else {
		shell = noopShell{}
	}
	var assets core.AssetServer
	if ctor := core.DefaultAssetServer(); ctor != nil {
		assets = ctor("")
	} else {
		assets = noopAssetServer{}
	}

	var notFoundRoute *core.Route
	if artifacts.GlobalNotFound != "" {
		notFoundRoute = &core.Route{Export: "GlobalNotFound", Pattern: "/*"}
	}
	// In embed mode, check for a pre-built stylesheet.
	var prebuiltCSS string
	if artifacts.CSSURL != "" {
		prebuiltCSS = artifacts.CSSURL
	}
	// Embed mode never supports hot reload — assets are baked into the binary.
	orch := NewOrchestrator(reg, artifacts.Routes, shell, assets, artifacts.BundleOutput, notFoundRoute, prebuiltCSS, false)
	app := newApp(cfg, reg, orch)
	app.SetArtifacts(artifacts.BundleOutput)
	return app, nil
}

// ── CSS import scanning ──────────────────────────────────────────────────────

// cssImportRE matches `import "foo.css"` or `import './bar.css'` in JS/TS files.
var cssImportRE = regexp.MustCompile(`(?m)^\s*import\s+["']([^"']+\.css)["']`)

// scanCSSImports walks appDir recursively, reads each source file matching
// exts, and extracts CSS import paths. Returns deduplicated absolute paths
// in discovery order.
// Note: uses os.ReadDir/os.ReadFile directly because the scanner works with
// absolute paths and core.FS implementations may prepend a root prefix.
func scanCSSImports(_ core.FS, appDir string, exts []string) []string {
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		extSet[e] = true
	}
	// Also scan .css extension itself so we don't miss anything.
	extSet[".css"] = false // present in map but we only scan source files for imports

	var cssFiles []string
	seen := map[string]bool{}

	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			path := filepath.Join(dir, name)
			if e.IsDir() {
				if name == "node_modules" || name == "public" || strings.HasPrefix(name, ".") {
					continue
				}
				walk(path)
				continue
			}
			ext := filepath.Ext(name)
			if !extSet[ext] {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			for scanner.Scan() {
				line := scanner.Text()
				// Stop scanning after the first non-import, non-blank, non-comment,
				// non-directive line — imports must be at the top.
				trimmed := strings.TrimSpace(line)
				if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if !strings.HasPrefix(trimmed, "import") && !strings.HasPrefix(trimmed, `"use `) && !strings.HasPrefix(trimmed, `'use `) {
					break
				}
				matches := cssImportRE.FindStringSubmatch(line)
				if len(matches) < 2 {
					continue
				}
				cssPath := matches[1]
				var abs string
				if strings.HasPrefix(cssPath, ".") {
					abs = filepath.Join(filepath.Dir(path), cssPath)
				} else {
					abs = filepath.Join(appDir, cssPath)
				}
				abs = filepath.Clean(abs)
				if !seen[abs] {
					seen[abs] = true
					cssFiles = append(cssFiles, abs)
				}
			}
		}
	}
	walk(filepath.Join(appDir, "app"))
	// Also scan components directory.
	walk(filepath.Join(appDir, "components"))
	return cssFiles
}

// generateCSSInput writes a temporary CSS file in appDir that @import's each
// discovered CSS file. The file is placed in appDir so the CSS processor
// (e.g. Tailwind) can resolve node_modules and scan content relative to the app.
// Returns the path and a cleanup function.
func generateCSSInput(cssFiles []string, appDir string) (string, func(), error) {
	var sb strings.Builder
	for _, f := range cssFiles {
		rel, err := filepath.Rel(appDir, f)
		if err != nil {
			rel = f
		}
		// Use ./ prefix for relative paths.
		if !strings.HasPrefix(rel, ".") {
			rel = "./" + rel
		}
		sb.WriteString(fmt.Sprintf("@import %q;\n", filepath.ToSlash(rel)))
	}

	tmpFile := filepath.Join(appDir, ".pola-css-input.css")
	if err := os.WriteFile(tmpFile, []byte(sb.String()), 0o644); err != nil {
		return "", func() {}, fmt.Errorf("write css input: %w", err)
	}
	return tmpFile, func() { _ = os.Remove(tmpFile) }, nil
}

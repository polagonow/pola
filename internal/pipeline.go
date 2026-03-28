package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	samberdo "github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

// prebuildMeta is the JSON schema for prebuild-meta.json written during mage Bundle
// and read back at startup in embed mode.
type prebuildMeta struct {
	Routes         []core.Route      `json:"routes"`
	ClientEntryURL string            `json:"clientEntryURL"`
	ImportURLs     map[string]string `json:"importURLs"`
	GlobalNotFound string            `json:"globalNotFound"`
	CSSURLs        []string          `json:"cssURLs,omitempty"`
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

func (noopShell) Render(_ core.ShellParams) string {
	return "<!DOCTYPE html><html><body></body></html>"
}

// newNotFoundRoute returns a catch-all Route for the global not-found page,
// or nil if globalNotFound is empty.
func newNotFoundRoute(globalNotFound string) *core.Route {
	if globalNotFound == "" {
		return nil
	}
	return &core.Route{Export: "GlobalNotFound", Pattern: "/*"}
}

// resolveShellAndAssets resolves HTMLShell and AssetServer from the DI container,
// falling back to noop implementations when none are registered.
func resolveShellAndAssets(injector samberdo.Injector, publicDir string) (core.HTMLShell, core.AssetServer) {
	var shell core.HTMLShell
	if s, err := samberdo.Invoke[core.HTMLShell](injector); err == nil {
		shell = s
	} else {
		shell = noopShell{}
	}
	var assets core.AssetServer
	if factory, err := samberdo.Invoke[core.AssetServerFactory](injector); err == nil {
		assets = factory(publicDir)
	} else {
		assets = noopAssetServer{}
	}
	return shell, assets
}

// Build runs the full build pipeline and returns a ready App.
func Build(cfg *core.Config) (*core.App, error) {
	// ── 0. Build DI container ───────────────────────────────────────────
	injector := di.Build()

	// ── 1. Resolve required services ────────────────────────────────────
	renderer, err := samberdo.Invoke[core.Renderer](injector)
	if err != nil {
		return nil, fmt.Errorf("pola: no Renderer registered; import _ \"github.com/polagonow/pola/renderer/react\"")
	}
	router, err := samberdo.Invoke[core.Router](injector)
	if err != nil {
		return nil, fmt.Errorf("pola: no Router registered; import _ \"github.com/polagonow/pola/router/nextjs\"")
	}
	// Bundler and FS are optional in embed/prebuild mode (no esbuild tag).
	bundler, _ := samberdo.Invoke[core.Bundler](injector)
	fsys, _ := samberdo.Invoke[core.FS](injector)
	logger, err := samberdo.Invoke[core.Logger](injector)
	if err != nil {
		return nil, fmt.Errorf("pola: no Logger registered; import _ \"github.com/polagonow/pola/logger/slog\"")
	}
	engine, err := samberdo.Invoke[core.JSEngine](injector)
	if err != nil {
		return nil, fmt.Errorf("pola: no JSEngine registered; import _ \"github.com/polagonow/pola/engine\" with a build tag (goja, quickjsgo, etc.)")
	}

	// Optional services — nil when not registered.
	metrics, _ := samberdo.Invoke[core.Metrics](injector)
	tracer, _ := samberdo.Invoke[core.Tracer](injector)
	css, _ := samberdo.Invoke[core.CSS](injector)
	pprof, _ := samberdo.Invoke[core.Pprof](injector)

	// Slice collectors
	middleware := samberdo.MustInvoke[*di.MiddlewareCollector](injector).All()
	injectors := samberdo.MustInvoke[*di.InjectorCollector](injector).All()

	// Propagate logger to all components that accept it.
	for _, c := range []any{engine, bundler, renderer, router, css} {
		if la, ok := c.(core.LogAware); ok {
			la.SetLogger(logger)
		}
	}

	// ── Prebuild fast-path (embed mode) ───────────────────────────────────
	if loader, err := samberdo.Invoke[core.PrebuildLoader](injector); err == nil {
		return buildFromPrebuilt(cfg, injector, loader, renderer, router, engine, logger, metrics, tracer, pprof, middleware, injectors)
	}

	// If no prebuild loader, bundler and FS are required.
	if bundler == nil {
		return nil, fmt.Errorf("pola: no Bundler registered; import _ \"github.com/polagonow/pola/bundler/esbuild\"")
	}
	if fsys == nil {
		return nil, fmt.Errorf("pola: no FS registered; import _ \"github.com/polagonow/pola/fs/osfs\"")
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
		publicDir = "./public"
	}
	publicDir, err = filepath.Abs(publicDir)
	if err != nil {
		return nil, fmt.Errorf("pola: resolve public dir: %w", err)
	}
	assetsURLPath := "/public/assets"

	ctx := context.Background()

	// ── 2. Scan routes ────────────────────────────────────────────────────
	exts := renderer.FileExtensions()
	routes, err := router.ScanRoutes(ctx, fsys, absWebAppPath, exts)
	if err != nil {
		return nil, fmt.Errorf("pola: scan routes: %w", err)
	}

	// ── 3. Get DiscoveryResult (optional, router may implement it) ────────
	var discovery core.DiscoveryResult
	if dp, ok := router.(discoveryProvider); ok {
		discovery = dp.DiscoveryResult()
	}

	// ── 4. Generate server entry source ───────────────────────────────────
	var serverEntryContent string
	if eg, ok := renderer.(entryGenerator); ok {
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
		CSSProcessor:       css, // may be nil — bundler stubs CSS when nil
	}
	if bc, ok := renderer.(bundleConfigurator); ok {
		bundleInput.ServerBundleConditions = bc.BundleConditions()
		bundleInput.ClientEntry = bc.ClientEntry()
	}

	bundleOutput, err := bundler.Build(ctx, bundleInput)
	if err != nil {
		return nil, fmt.Errorf("pola: bundle: %w", err)
	}

	// CSS URLs are reported by the bundler (emitted as hashed output files).
	var cssURLs []string
	if bundleOutput != nil {
		cssURLs = bundleOutput.CSSURLs
	}

	// ── 6.5. Wire bundle to renderer ─────────────────────────────────────
	if bl, ok := renderer.(core.BundleLoader); ok && engine != nil && bundleOutput != nil {
		if err := bl.LoadBundle(engine, bundleOutput.ServerBundle); err != nil {
			return nil, fmt.Errorf("pola: load bundle: %w", err)
		}
	}

	// ── 7. Resolve shell and asset server ─────────────────────────────────
	shell, assets := resolveShellAndAssets(injector, publicDir)

	// ── 8. Wire orchestrator ───────────────────────────────────────────────
	notFoundRoute := newNotFoundRoute(discovery.GlobalNotFound)
	orch := NewOrchestrator(renderer, router, logger, metrics, tracer, pprof, middleware, injectors, routes, shell, assets, bundleOutput, notFoundRoute, cssURLs, cfg.Dev)

	// ── 6.6. Copy static public files ────────────────────────────────────────
	srcPublic := filepath.Join(absWebAppPath, "public")
	if srcPublic != publicDir {
		copyPublicStatics(srcPublic, publicDir)
	}

	// ── 9. Write prebuild-meta.json for embed builds ───────────────────────
	if bundleOutput != nil {
		meta := prebuildMeta{
			Routes:         routes,
			ClientEntryURL: bundleOutput.ClientEntryURL,
			ImportURLs:     bundleOutput.ImportURLs,
			GlobalNotFound: discovery.GlobalNotFound,
			CSSURLs:        bundleOutput.CSSURLs,
		}
		if b, err := json.Marshal(meta); err == nil {
			_ = os.WriteFile(filepath.Join(publicDir, "prebuild-meta.json"), b, 0o644)
		}
	}

	// ── 10. Return App ─────────────────────────────────────────────────────
	app := newApp(cfg, injector, orch)
	app.SetArtifacts(bundleOutput)

	// In dev mode wrap the app in a hot-reloader so file changes trigger a
	// full rebuild and browser reload via WebSocket.
	if cfg.Dev {
		hr, err := NewHotReloader(cfg, injector, app, notFoundRoute)
		if err != nil {
			return nil, fmt.Errorf("pola: hotreload: %w", err)
		}
		devApp := newApp(cfg, injector, hr.Handler())
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
func buildFromPrebuilt(
	cfg *core.Config,
	injector samberdo.Injector,
	loader core.PrebuildLoader,
	renderer core.Renderer,
	router core.Router,
	engine core.JSEngine,
	logger core.Logger,
	metrics core.Metrics,
	tracer core.Tracer,
	pprof core.Pprof,
	middleware []core.Middleware,
	injectors []core.RuntimeInjector,
) (*core.App, error) {
	artifacts, err := loader()
	if err != nil {
		return nil, fmt.Errorf("pola: load prebuilt: %w", err)
	}

	// Seed the router's internal routing table from the pre-built route list.
	if rl, ok := router.(routeLoader); ok {
		rl.LoadRoutes(artifacts.Routes)
	}

	// Load the server bundle into the JS engine.
	if bl, ok := renderer.(core.BundleLoader); ok && engine != nil && artifacts.BundleOutput != nil {
		if err := bl.LoadBundle(engine, artifacts.BundleOutput.ServerBundle); err != nil {
			return nil, fmt.Errorf("pola: load bundle: %w", err)
		}
	}

	// Resolve shell and asset server (pola_embed.go is injected via overlay during builds).
	shell, assets := resolveShellAndAssets(injector, "")

	notFoundRoute := newNotFoundRoute(artifacts.GlobalNotFound)
	orch := NewOrchestrator(renderer, router, logger, metrics, tracer, pprof, middleware, injectors, artifacts.Routes, shell, assets, artifacts.BundleOutput, notFoundRoute, artifacts.CSSURLs, false)
	app := newApp(cfg, injector, orch)
	app.SetArtifacts(artifacts.BundleOutput)
	return app, nil
}

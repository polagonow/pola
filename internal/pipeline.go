package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/polagonow/pola/core"
)

// prebuildMeta is the JSON schema for prebuild-meta.json written during mage Bundle
// and read back at startup in embed mode.
type prebuildMeta struct {
	Routes         []core.Route        `json:"routes"`
	ClientEntryURL string              `json:"clientEntryURL"`
	ImportURLs     map[string]string   `json:"importURLs"`
	GlobalNotFound string              `json:"globalNotFound"`
	CSSURLs        []string            `json:"cssURLs,omitempty"`
	DocumentProps  *core.DocumentProps `json:"documentProps,omitempty"`
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

// bundleDefiner is the optional interface a Renderer may implement to provide
// additional esbuild defines for the server bundle (e.g. __CLIENT_MANIFEST__).
type bundleDefiner interface {
	BundleDefines() map[string]string
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

// resolveShellAndAssets resolves HTMLShell and AssetServer from the registry,
// falling back to noop implementations when none are registered.
func resolveShellAndAssets(registry *core.Registry, publicDir string) (core.HTMLShell, core.AssetServer) {
	var shell core.HTMLShell
	if s, err := core.Invoke[core.HTMLShell](registry); err == nil {
		shell = s
	} else {
		shell = noopShell{}
	}
	var assets core.AssetServer
	if factory, err := core.Invoke[core.AssetServerFactory](registry); err == nil {
		assets = factory(publicDir)
	} else {
		assets = noopAssetServer{}
	}
	return shell, assets
}

// wireRenderDeps calls SetRenderDeps on the renderer if it implements
// core.RenderDepsAware, providing all framework-level dependencies.
func wireRenderDeps(renderer core.Renderer, deps core.RenderDeps) {
	if rda, ok := renderer.(core.RenderDepsAware); ok {
		rda.SetRenderDeps(deps)
	}
}

// Build runs the full build pipeline from an AppBuilder.
func Build(ctx context.Context, builder *core.AppBuilder) (*core.App, error) {
	registry, err := builder.BuildRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("pola: build registry: %w", err)
	}

	// Register framework-internal singletons needed by hotreload.
	core.Provide[*EventBus](registry, func() (*EventBus, error) {
		return NewEventBus(), nil
	})

	return buildWithRegistry(builder.Config(), registry)
}

// buildWithRegistry is the core pipeline implementation.
func buildWithRegistry(cfg *core.Config, registry *core.Registry) (*core.App, error) {
	// ── 1. Resolve required services ────────────────────────────────────
	renderer, err := core.Invoke[core.Renderer](registry)
	if err != nil {
		return nil, fmt.Errorf("pola: no Renderer registered — register a renderer plugin (e.g. react.Plugin())")
	}
	router, err := core.Invoke[core.Router](registry)
	if err != nil {
		return nil, fmt.Errorf("pola: no Router registered — use nextjs.Plugin()")
	}
	// Bundler and FS are optional in embed/prebuild mode.
	bundler, _ := core.Invoke[core.Bundler](registry)
	fsys, _ := core.Invoke[core.FS](registry)
	logger, err := core.Invoke[core.Logger](registry)
	if err != nil {
		return nil, fmt.Errorf("pola: no Logger registered — use slog.Plugin()")
	}
	engine, err := core.Invoke[core.JSEngine](registry)
	if err != nil {
		return nil, fmt.Errorf("pola: no JSEngine registered — use goja.Plugin()")
	}

	// Optional services — nil when not registered.
	metrics, _ := core.Invoke[core.Metrics](registry)
	tracer, _ := core.Invoke[core.Tracer](registry)
	css, _ := core.Invoke[core.CSS](registry)
	pprof, _ := core.Invoke[core.Pprof](registry)

	// Middleware and injectors from Registry.
	middleware := registry.Middleware()
	runtimeInjectors := registry.RuntimeInjectors()

	// ── 1.5. Build API routes ───────────────────────────────────────────
	var apiRouter core.APIRouter
	if ar, err := core.Invoke[core.APIRouter](registry); err == nil {
		type builder interface {
			Build(*core.Registry) error
		}
		if b, ok := ar.(builder); ok {
			if buildErr := b.Build(registry); buildErr != nil {
				return nil, fmt.Errorf("pola: build api routes: %w", buildErr)
			}
		}
		apiRouter = ar
	}

	// Propagate logger to all components that accept it.
	for _, c := range []any{engine, bundler, renderer, router, css} {
		if la, ok := c.(core.LogAware); ok {
			la.SetLogger(logger)
		}
	}

	// ── Prebuild fast-path (embed mode) ───────────────────────────────────
	if loader, err := core.Invoke[core.PrebuildLoader](registry); err == nil {
		return buildFromPrebuilt(cfg, registry, loader, renderer, router, engine, logger, metrics, tracer, pprof, middleware, runtimeInjectors)
	}

	// If no prebuild loader, bundler and FS are required.
	if bundler == nil {
		return nil, fmt.Errorf("pola: no Bundler registered — use esbuild.Plugin()")
	}
	if fsys == nil {
		return nil, fmt.Errorf("pola: no FS registered — use osfs.Plugin()")
	}

	// ── Resolve paths ─────────────────────────────────────────────────────
	webAppPath := cfg.WebAppPath
	if webAppPath == "" {
		webAppPath = "./web"
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
		WatchExtensions:    mergeWatchExtensions(renderer.FileExtensions()),
	}
	if bc, ok := renderer.(bundleConfigurator); ok {
		bundleInput.ServerBundleConditions = bc.BundleConditions()
		bundleInput.ClientEntry = bc.ClientEntry()
	}
	if bd, ok := renderer.(bundleDefiner); ok {
		bundleInput.ServerBundleDefines = bd.BundleDefines()
	}
	if bp, err := core.Invoke[core.BundlePluginProvider](registry); err == nil {
		bundleInput.ClientPlugins = bp.ClientPlugins(absWebAppPath)
		bundleInput.ServerPlugins = bp.ServerPlugins(absWebAppPath)
		bundleInput.ProbePlugins = bp.ProbePlugins(absWebAppPath)
		bundleInput.ClientModuleStub = bp.ClientModuleStub()
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

	// ── 6.6. Extract document props from root layout ────────────────────
	var docProps *core.DocumentProps
	if extractor, ok := renderer.(interface {
		ExtractDocumentProps() (*core.DocumentProps, error)
	}); ok {
		dp, err := extractor.ExtractDocumentProps()
		if err != nil {
			logger.Warn("pola: extract document props", "err", err)
		} else {
			docProps = dp
		}
	}

	// ── 7. Resolve shell and asset server ─────────────────────────────────
	shell, assets := resolveShellAndAssets(registry, publicDir)

	// ── 8. Wire render deps to renderer ───────────────────────────────────
	renderCache, cacheErr := core.Invoke[core.Cache](registry)
	if cacheErr != nil {
		logger.Info("pola: cache not registered, caching disabled")
	}

	notFoundRoute := newNotFoundRoute(discovery.GlobalNotFound)

	var devScript string
	if cfg.Dev {
		devScript = ClientScript
	}

	wireRenderDeps(renderer, core.RenderDeps{
		Shell:         shell,
		Cache:         renderCache,
		Logger:        logger,
		Metrics:       metrics,
		Tracer:        tracer,
		BundleOutput:  bundleOutput,
		CSSURLs:       cssURLs,
		DocumentProps: docProps,
		DevScript:     devScript,
		NotFoundRoute: notFoundRoute,
	})

	// ── 9. Wire orchestrator ──────────────────────────────────────────────
	// Wrap injectors with per-request memoization to deduplicate Go calls.
	memoInjectors := WrapInjectorsWithMemo(runtimeInjectors)

	orch := NewOrchestrator(renderer, router, apiRouter, logger, metrics, tracer, pprof, middleware, memoInjectors, routes, assets, cfg.Dev)

	// ── Copy static public files ────────────────────────────────────────
	srcPublic := filepath.Join(absWebAppPath, "public")
	if srcPublic != publicDir {
		copyPublicStatics(srcPublic, publicDir)
	}

	// ── 10. Write prebuild-meta.json for embed builds ───────────────────────
	if bundleOutput != nil {
		meta := prebuildMeta{
			Routes:         routes,
			ClientEntryURL: bundleOutput.ClientEntryURL,
			ImportURLs:     bundleOutput.ImportURLs,
			GlobalNotFound: discovery.GlobalNotFound,
			CSSURLs:        bundleOutput.CSSURLs,
			DocumentProps:  docProps,
		}
		if b, err := json.Marshal(meta); err == nil {
			_ = os.WriteFile(filepath.Join(publicDir, "prebuild-meta.json"), b, 0o644)
		}
	}

	// ── 11. Return App ─────────────────────────────────────────────────────
	app := newApp(cfg, registry, orch)
	app.SetArtifacts(bundleOutput)

	// In dev mode wrap the app in a hot-reloader so file changes trigger a
	// full rebuild and browser reload via WebSocket.
	if cfg.Dev {
		hr, err := NewHotReloader(cfg, registry, app, notFoundRoute)
		if err != nil {
			return nil, fmt.Errorf("pola: hotreload: %w", err)
		}
		devApp := newApp(cfg, registry, hr.Handler())
		devApp.SetArtifacts(bundleOutput)
		return devApp, nil
	}

	return app, nil
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

// mergeWatchExtensions combines the renderer's file extensions with framework
// defaults (.css) into a deduplicated list for the bundler's watch mode.
func mergeWatchExtensions(rendererExts []string) []string {
	seen := make(map[string]bool, len(rendererExts)+1)
	var result []string
	for _, ext := range rendererExts {
		if !seen[ext] {
			result = append(result, ext)
			seen[ext] = true
		}
	}
	if !seen[".css"] {
		result = append(result, ".css")
	}
	return result
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
	registry *core.Registry,
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

	// Resolve shell and asset server.
	shell, assets := resolveShellAndAssets(registry, "")

	// Resolve cache for render result caching; nil when cache is disabled.
	renderCache, cacheErr := core.Invoke[core.Cache](registry)
	if cacheErr != nil {
		logger.Info("pola: cache not registered, caching disabled")
	}

	notFoundRoute := newNotFoundRoute(artifacts.GlobalNotFound)

	wireRenderDeps(renderer, core.RenderDeps{
		Shell:         shell,
		Cache:         renderCache,
		Logger:        logger,
		Metrics:       metrics,
		Tracer:        tracer,
		BundleOutput:  artifacts.BundleOutput,
		CSSURLs:       artifacts.CSSURLs,
		DocumentProps: artifacts.DocumentProps,
		NotFoundRoute: notFoundRoute,
	})

	// Wrap injectors with per-request memoization.
	memoInjectors := WrapInjectorsWithMemo(injectors)

	// In prebuild/embed mode, API routes are still compiled into the binary.
	var apiRouter core.APIRouter
	if ar, err := core.Invoke[core.APIRouter](registry); err == nil {
		type builder interface {
			Build(*core.Registry) error
		}
		if b, ok := ar.(builder); ok {
			if buildErr := b.Build(registry); buildErr != nil {
				return nil, fmt.Errorf("pola: build api routes: %w", buildErr)
			}
		}
		apiRouter = ar
	}
	orch := NewOrchestrator(renderer, router, apiRouter, logger, metrics, tracer, pprof, middleware, memoInjectors, artifacts.Routes, assets, false)
	app := newApp(cfg, registry, orch)
	app.SetArtifacts(artifacts.BundleOutput)
	return app, nil
}

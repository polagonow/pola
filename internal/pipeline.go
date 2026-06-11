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
	"github.com/polagonow/pola/core/reserved"
	"github.com/polagonow/pola/mailer"
	reactrenderer "github.com/polagonow/pola/mailer/renderer/react"
	"github.com/polagonow/pola/routes"
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
	ServerActions  map[string][]string `json:"serverActions,omitempty"`
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

// serverActionStubProvider is the optional interface a BundlePluginProvider may
// implement to supply the client-side stub generator for 'use server' modules.
type serverActionStubProvider interface {
	ServerActionStub() func(absPath, moduleID string, exports []string) string
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
	if cfg.APIOnly {
		return buildAPIOnly(cfg, registry)
	}

	// ── 1. Resolve required services ────────────────────────────────────
	renderer, err := core.Invoke[core.Renderer](registry)
	if err != nil {
		// No renderer registered — this is an API-only app.
		return buildAPIOnly(cfg, registry)
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
	assetsURLPath := reserved.Assets

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
		// Optional: providers that support 'use server' modules supply a stub
		// generator for the client bundle.
		if sp, ok := bp.(serverActionStubProvider); ok {
			bundleInput.ServerActionStub = sp.ServerActionStub()
		}
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

	// ── 6.55. Build email templates bundle (if mailers exist) ───────────
	buildEmailBundle(ctx, absWebAppPath, renderer, bundler, engine, logger, registry)

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

	orch := NewOrchestrator(renderer, router, apiRouter, logger, metrics, tracer, pprof, renderCache, middleware, memoInjectors, routes, assets, cfg.Dev)
	orch.SetServerActionHandlers(newServerActionHandler(renderer, bundleOutput, memoInjectors, renderCache, cfg.Dev, logger))

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
			ServerActions:  bundleOutput.ServerActions,
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
	orch := NewOrchestrator(renderer, router, apiRouter, logger, metrics, tracer, pprof, renderCache, middleware, memoInjectors, artifacts.Routes, assets, false)
	orch.SetServerActionHandlers(newServerActionHandler(renderer, artifacts.BundleOutput, memoInjectors, renderCache, false, logger))
	app := newApp(cfg, registry, orch)
	app.SetArtifacts(artifacts.BundleOutput)
	return app, nil
}

// buildEmailBundle checks for email templates under appDir/mailers/ and, if
// found, loads them into the registered mailer renderer. For filesystem-based
// renderers (e.g. tmpl) it calls LoadTemplates directly; for the react
// renderer it generates a TypeScript entry, bundles it, and loads the bundle.
func buildEmailBundle(
	ctx context.Context,
	appDir string,
	renderer core.Renderer,
	bundler core.Bundler,
	engine core.JSEngine,
	logger core.Logger,
	registry *core.Registry,
) {
	mailersDir := filepath.Join(appDir, "mailers")
	if _, err := os.Stat(mailersDir); os.IsNotExist(err) {
		return
	}

	// Try filesystem-based template loader first (e.g. tmpl renderer).
	if loader, err := core.Invoke[mailer.TemplateLoader](registry); err == nil {
		if err := loader.LoadTemplates(mailersDir); err != nil {
			logger.Warn("pola: load email templates", "err", err)
		}
		return
	}

	// Fall back to react email bundle path.
	emailRenderer, err := core.Invoke[*reactrenderer.Renderer](registry)
	if err != nil {
		return // mailer plugin not registered — nothing to do
	}

	exts := renderer.FileExtensions()
	templates, layouts, err := mailer.ScanMailers(mailersDir, exts)
	if err != nil {
		logger.Warn("pola: scan mailers", "err", err)
		return
	}
	if len(templates) == 0 {
		return
	}

	// Generate the email entry TypeScript source.
	entrySource := mailer.GenerateEmailEntry(templates, layouts, mailersDir)

	// Bundle the email entry using a temp output directory.
	tmpDir, err := os.MkdirTemp("", "pola-email-bundle-*")
	if err != nil {
		logger.Warn("pola: email bundle tmpdir", "err", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	emailBundleInput := core.BundleInput{
		AppDir:                 appDir,
		OutDir:                 filepath.Join(tmpDir, "out"),
		ServerEntryContent:     entrySource,
		ServerBundleConditions: []string{"module", "default"},
	}

	output, err := bundler.Build(ctx, emailBundleInput)
	if err != nil {
		logger.Warn("pola: bundle email templates", "err", err)
		return
	}
	if output == nil || len(output.ServerBundle) == 0 {
		logger.Warn("pola: email bundle produced no output")
		return
	}

	if err := emailRenderer.LoadBundle(output.ServerBundle); err != nil {
		logger.Warn("pola: load email bundle", "err", err)
	} else {
		logger.Info("pola: email templates loaded", "templates", len(templates), "layouts", len(layouts))
	}
}

// buildAPIOnly constructs an App for API-only projects (no renderer/router/bundler/engine).
func buildAPIOnly(cfg *core.Config, registry *core.Registry) (*core.App, error) {
	logger, err := core.Invoke[core.Logger](registry)
	if err != nil {
		return nil, fmt.Errorf("pola: no Logger registered — use slog.Plugin()")
	}

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

	metrics, _ := core.Invoke[core.Metrics](registry)
	tracer, _ := core.Invoke[core.Tracer](registry)
	pprofSvc, _ := core.Invoke[core.Pprof](registry)
	middleware := registry.Middleware()

	logger.Info("pola: API-only mode (no renderer/bundler)")

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if tracer != nil {
			var span core.Span
			ctx, span = tracer.StartSpan(ctx, "pola.api")
			defer span.End()
			r = r.WithContext(ctx)
		}
		if metrics != nil && r.URL.Path == metrics.Path() {
			metrics.Handler().ServeHTTP(w, r)
			return
		}
		if pprofSvc != nil && strings.HasPrefix(r.URL.Path, pprofSvc.Path()) {
			pprofSvc.Handler().ServeHTTP(w, r)
			return
		}
		if apiRouter != nil {
			if h, params, ok := apiRouter.Match(r); ok {
				if params != nil {
					r = routes.WithParams(r, params)
				}
				h(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})

	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i].Wrap(handler)
	}

	return newApp(cfg, registry, handler), nil
}

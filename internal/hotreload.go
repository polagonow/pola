package internal

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/polagonow/pola/cache/memory"
	"github.com/polagonow/pola/core"
)

// ClientScript is the browser-side WebSocket listener that triggers a page
// reload when the dev server rebuilds.
const ClientScript = `(function(){` +
	`function connect(){` +
	`var ws=new WebSocket("ws://"+location.host+"/__dev__/hot");` +
	`ws.onmessage=function(e){if(e.data==="reload")location.reload()};` +
	`ws.onclose=function(){setTimeout(connect,2000)};` +
	`}connect();` +
	`})();`

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: checkSameOrigin,
}

// checkSameOrigin validates that the WebSocket Origin header matches the
// request Host, preventing cross-origin WebSocket hijacking.
func checkSameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// liveApp pairs a built App with its http.Handler.
type liveApp struct {
	app     *core.App
	handler http.Handler
}

// HotReloader watches for bundler rebuild events and notifies browsers
// via WebSocket.
type HotReloader struct {
	cfg           *core.Config
	registry      *core.Registry
	bus           *EventBus
	notFoundRoute *core.Route
	current       atomic.Pointer[liveApp]
	done          chan struct{}
}

// NewHotReloader creates a HotReloader for the given config and initial app.
// It listens on the bundler's Watch channel for rebuild events and notifies
// connected browsers via WebSocket.
func NewHotReloader(cfg *core.Config, registry *core.Registry, initial *core.App, notFoundRoute *core.Route) (*HotReloader, error) {
	bus := core.MustInvoke[*EventBus](registry)
	logger := core.MustInvoke[core.Logger](registry)

	h := &HotReloader{
		cfg:           cfg,
		registry:      registry,
		bus:           bus,
		notFoundRoute: notFoundRoute,
		done:          make(chan struct{}),
	}

	live := &liveApp{app: initial, handler: initial}
	h.current.Store(live)

	// Start watching via bundler's Watch channel.
	bundler, err := core.Invoke[core.Bundler](registry)
	if err != nil {
		// No bundler — fall back to no-op watch.
		return h, nil
	}

	// Reconstruct the bundle input for watch mode.
	renderer, _ := core.Invoke[core.Renderer](registry)
	router, _ := core.Invoke[core.Router](registry)
	css, _ := core.Invoke[core.CSS](registry)
	engine, _ := core.Invoke[core.JSEngine](registry)
	fsys, _ := core.Invoke[core.FS](registry)

	output := initial.Artifacts().Output
	if output == nil {
		return h, nil
	}

	// Use the same bundle input shape, derived from config.
	webAppPath := cfg.WebAppPath
	if webAppPath == "" {
		webAppPath = "./app"
	}
	publicDir := cfg.PublicDir
	if publicDir == "" {
		publicDir = "./public"
	}
	absWebAppPath, _ := filepath.Abs(webAppPath)
	absPublicDir, _ := filepath.Abs(publicDir)

	// Discover client components and generate server entry — these are
	// required for the bundler to produce a working ClientEntryURL and
	// ServerBundle. Without them the watch rebuild produces empty output.
	var clientComponents []string
	var serverEntryContent string
	if router != nil && fsys != nil && renderer != nil {
		exts := renderer.FileExtensions()
		if _, scanErr := router.ScanRoutes(context.Background(), fsys, absWebAppPath, exts); scanErr == nil {
			if dp, ok := router.(interface {
				DiscoveryResult() core.DiscoveryResult
			}); ok {
				disc := dp.DiscoveryResult()
				clientComponents = disc.ClientComponents
			}
			if eg, ok := renderer.(interface {
				GenerateEntry(core.DiscoveryResult) (string, error)
			}); ok {
				if dp, ok2 := router.(interface {
					DiscoveryResult() core.DiscoveryResult
				}); ok2 {
					serverEntryContent, _ = eg.GenerateEntry(dp.DiscoveryResult())
				}
			}
		}
	}

	bundleInput := core.BundleInput{
		AppDir:             absWebAppPath,
		OutDir:             filepath.Join(absPublicDir, "assets"),
		AssetsURLPath:      "/public/assets",
		ClientComponents:   clientComponents,
		ServerEntryContent: serverEntryContent,
		Dev:                true,
		CSSProcessor:       css,
	}
	if renderer != nil {
		bundleInput.WatchExtensions = mergeWatchExtensions(renderer.FileExtensions())
	}
	// Add renderer-specific bundle configuration (conditions, client entry).
	if bc, ok := renderer.(interface {
		BundleConditions() []string
		ClientEntry() string
	}); ok {
		bundleInput.ServerBundleConditions = bc.BundleConditions()
		bundleInput.ClientEntry = bc.ClientEntry()
	}
	// Add bundler-specific plugins from the plugin provider.
	if bp, err := core.Invoke[core.BundlePluginProvider](registry); err == nil {
		bundleInput.ClientPlugins = bp.ClientPlugins(absWebAppPath)
		bundleInput.ServerPlugins = bp.ServerPlugins(absWebAppPath)
		bundleInput.ProbePlugins = bp.ProbePlugins(absWebAppPath)
		bundleInput.ClientModuleStub = bp.ClientModuleStub()
	}

	watchCh, watchErr := bundler.Watch(h.contextFromDone(), bundleInput)
	if watchErr != nil {
		logger.Warn("hotreload: bundler watch not available", "err", watchErr)
		return h, nil
	}

	// Goroutine: read from bundler watch channel.
	go func() {
		for {
			select {
			case <-h.done:
				return
			case newOutput, ok := <-watchCh:
				if !ok {
					return
				}
				logger.Info("hotreload: bundler delivered new output")

				// Rewire: load bundle into renderer if applicable.
				if bl, ok := renderer.(core.BundleLoader); ok && engine != nil && newOutput != nil {
					if err := bl.LoadBundle(engine, newOutput.ServerBundle); err != nil {
						logger.Error("hotreload: load bundle", "err", err)
						continue
					}
				}

				// Rebuild orchestrator with new output.
				metrics, _ := core.Invoke[core.Metrics](registry)
				tracer, _ := core.Invoke[core.Tracer](registry)
				pprof, _ := core.Invoke[core.Pprof](registry)

				mws := registry.Middleware()
				injs := registry.RuntimeInjectors()
				shell, assets := resolveShellAndAssets(registry, publicDir)

				var cssURLs []string
				if newOutput != nil {
					cssURLs = newOutput.CSSURLs
				}

				// Re-extract document props from the (potentially updated) root layout.
				var docProps *core.DocumentProps
				if extractor, ok := renderer.(interface {
					ExtractDocumentProps() (*core.DocumentProps, error)
				}); ok {
					docProps, _ = extractor.ExtractDocumentProps()
				}

				// Resolve API router for hot-reload rebuild.
				var apiRouter core.APIRouter
				if ar, err := core.Invoke[core.APIRouter](registry); err == nil {
					apiRouter = ar
				}
				renderCache, err := core.Invoke[core.Cache](registry)
				if err != nil {
					renderCache = memory.MustNew(0)
				}
				// Clear stale SSR data from previous build.
				if err := renderCache.Clear(context.Background()); err != nil {
					logger.Warn("hotreload: cache clear", "err", err)
				}
				orch := NewOrchestrator(renderer, router, apiRouter, logger, metrics, tracer, pprof, renderCache, mws, WrapInjectorsWithMemo(injs), nil, shell, assets, newOutput, h.notFoundRoute, cssURLs, docProps, true)

				newApp := newApp(cfg, registry, orch)
				newApp.SetArtifacts(newOutput)
				h.current.Store(&liveApp{app: newApp, handler: newApp})

				logger.Info("hotreload: rebuild complete")
				bus.Publish("update", []byte("reload"))
			}
		}
	}()

	return h, nil
}

// contextFromDone returns a context.Context that cancels when h.done is closed.
// The caller must ensure Close() is called to avoid a goroutine leak.
func (h *HotReloader) contextFromDone() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-h.done
		cancel()
	}()
	return ctx
}

// Handler returns an http.Handler that serves WebSocket at /__dev__/hot
// and delegates all other requests to the current live App.
func (h *HotReloader) Handler() http.Handler {
	ws := &wsServer{bus: h.bus}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__dev__/hot" {
			ws.ServeHTTP(w, r)
			return
		}
		h.current.Load().handler.ServeHTTP(w, r)
	})
}

// Close stops watching and pending operations.
func (h *HotReloader) Close() error {
	close(h.done)
	return nil
}

// ── WebSocket server ──────────────────────────────────────────────────────────

type wsServer struct {
	bus *EventBus
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	sub := s.bus.Subscribe("update")
	defer sub.Close()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-sub.Wait():
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

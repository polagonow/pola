package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
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
	injector      do.Injector
	bus           *di.EventBus
	notFoundRoute *core.Route
	current       atomic.Pointer[liveApp]
	done          chan struct{}
}

// NewHotReloader creates a HotReloader for the given config and initial app.
// It listens on the bundler's Watch channel for rebuild events and notifies
// connected browsers via WebSocket.
func NewHotReloader(cfg *core.Config, injector do.Injector, initial *core.App, notFoundRoute *core.Route) (*HotReloader, error) {
	bus := do.MustInvoke[*di.EventBus](injector)
	logger := do.MustInvoke[core.Logger](injector)

	h := &HotReloader{
		cfg:           cfg,
		injector:      injector,
		bus:           bus,
		notFoundRoute: notFoundRoute,
		done:          make(chan struct{}),
	}

	live := &liveApp{app: initial, handler: initial}
	h.current.Store(live)

	// Start watching via bundler's Watch channel.
	bundler, err := do.Invoke[core.Bundler](injector)
	if err != nil {
		// No bundler — fall back to no-op watch.
		return h, nil
	}

	// Reconstruct the bundle input for watch mode.
	renderer, _ := do.Invoke[core.Renderer](injector)
	router, _ := do.Invoke[core.Router](injector)
	fsys, _ := do.Invoke[core.FS](injector)
	css, _ := do.Invoke[core.CSS](injector)
	engine, _ := do.Invoke[core.JSEngine](injector)
	_ = router
	_ = fsys

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

	bundleInput := core.BundleInput{
		AppDir:        webAppPath,
		OutDir:        publicDir + "/assets",
		AssetsURLPath: "/public/assets",
		Dev:           true,
		CSSProcessor:  css,
	}

	watchCh, watchErr := bundler.Watch(h.contextFromDone(), bundleInput)
	if watchErr != nil {
		logger.Warn("hotreload: bundler watch not available, falling back to full rebuild", "err", watchErr)
		// Fall back to FS-based watching if bundler.Watch is not implemented.
		if fsys != nil {
			return h.fallbackFSWatch(cfg, logger, fsys, renderer)
		}
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
				metrics, _ := do.Invoke[core.Metrics](injector)
				tracer, _ := do.Invoke[core.Tracer](injector)
				pprof, _ := do.Invoke[core.Pprof](injector)
				mws := do.MustInvoke[*di.MiddlewareCollector](injector).All()
				injs := do.MustInvoke[*di.InjectorCollector](injector).All()
				shell, assets := resolveShellAndAssets(injector, publicDir)

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

				orch := NewOrchestrator(renderer, router, logger, metrics, tracer, pprof, mws, injs, nil, shell, assets, newOutput, h.notFoundRoute, cssURLs, docProps, true)

				newApp := newApp(cfg, injector, orch)
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
func (h *HotReloader) contextFromDone() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-h.done
		cancel()
	}()
	return ctx
}

// fallbackFSWatch sets up FS-based file watching when the bundler doesn't
// support Watch. It does a full rebuild on file changes.
func (h *HotReloader) fallbackFSWatch(cfg *core.Config, logger core.Logger, fsys core.FS, renderer core.Renderer) (*HotReloader, error) {
	var exts []string
	if renderer != nil {
		exts = renderer.FileExtensions()
	}
	exts = append(exts, ".css")

	webAppPath := cfg.WebAppPath
	if webAppPath == "" {
		webAppPath = "./app"
	}

	if err := fsys.Watch(webAppPath, func(path string) {
		if !isSourceFile(path, webAppPath, exts) {
			return
		}
		logger.Info("hotreload: file changed, rebuilding", "path", path)
		newApp, err := Build(cfg)
		if err != nil {
			logger.Error("hotreload: build error", "err", err)
			return
		}
		h.current.Store(&liveApp{app: newApp, handler: newApp})
		logger.Info("hotreload: rebuild complete")
		h.bus.Publish("update", []byte("reload"))
	}); err != nil {
		return nil, fmt.Errorf("hotreload: fs watch: %w", err)
	}

	return h, nil
}

// isSourceFile returns true when path has one of the watched extensions and
// is not inside an excluded directory.
func isSourceFile(path, appDir string, exts []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	matched := false
	for _, e := range exts {
		if ext == e {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	rel, err := filepath.Rel(appDir, path)
	if err != nil {
		return false
	}
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == "node_modules" || seg == "public" || strings.HasPrefix(seg, ".") {
			return false
		}
	}
	return true
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
	bus *di.EventBus
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

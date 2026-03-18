package internal

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

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
	CheckOrigin: func(r *http.Request) bool { return true },
}

// liveApp pairs a built App with its http.Handler.
type liveApp struct {
	app     *core.App
	handler http.Handler
}

// HotReloader watches the app directory for source file changes, rebuilds,
// and notifies browsers via WebSocket.
type HotReloader struct {
	cfg     *core.Config
	reg     *core.Registry
	exts    []string // file extensions to watch, from renderer
	appDir  string   // absolute path
	bus     *memoryBus
	current atomic.Pointer[liveApp]
	timerMu sync.Mutex
	timer   *time.Timer
	building atomic.Bool
	done    chan struct{}
}

// NewHotReloader creates a HotReloader for the given config and initial app.
// It watches cfg.WebAppPath for changes to files matching the renderer's
// FileExtensions, rebuilds via internal.Build, and notifies browsers.
func NewHotReloader(cfg *core.Config, initial *core.App) (*HotReloader, error) {
	reg := cfg.Registry
	if reg == nil {
		reg = &core.Registry{}
	}

	// Determine file extensions to watch from the renderer (if available).
	var exts []string
	if reg.Renderer != nil {
		exts = reg.Renderer.FileExtensions()
	}
	// Always watch CSS too.
	exts = append(exts, ".css")

	webAppPath := cfg.WebAppPath
	if webAppPath == "" {
		webAppPath = "./app"
	}
	absAppDir, err := filepath.Abs(webAppPath)
	if err != nil {
		return nil, fmt.Errorf("hotreload: abs app dir: %w", err)
	}

	h := &HotReloader{
		cfg:    cfg,
		reg:    reg,
		exts:   exts,
		appDir: absAppDir,
		bus:    newMemoryBus(),
		done:   make(chan struct{}),
	}

	live := &liveApp{app: initial, handler: initial}
	h.current.Store(live)

	// Start FS watch via the registry's FS.Watch (if available).
	if reg.FS != nil {
		if err := reg.FS.Watch(absAppDir, h.onFileChange); err != nil {
			return nil, fmt.Errorf("hotreload: fs watch: %w", err)
		}
	}

	return h, nil
}

// Handler returns an http.Handler that serves WebSocket at /__dev__/hot
// and delegates all other requests to the current live App.
func (h *HotReloader) Handler() http.Handler {
	ws := &wsServer{ps: h.bus}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__dev__/hot" {
			ws.ServeHTTP(w, r)
			return
		}
		h.current.Load().handler.ServeHTTP(w, r)
	})
}

// Close stops pending rebuild timers.
func (h *HotReloader) Close() error {
	h.timerMu.Lock()
	if h.timer != nil {
		h.timer.Stop()
	}
	h.timerMu.Unlock()
	close(h.done)
	return nil
}

// onFileChange is called by FS.Watch for every changed path.
func (h *HotReloader) onFileChange(path string) {
	if !h.isSourceFile(path) {
		return
	}
	h.scheduleRebuild()
}

// isSourceFile returns true when path has one of the watched extensions and
// is not inside an excluded directory (node_modules, public, hidden dirs).
func (h *HotReloader) isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	matched := false
	for _, e := range h.exts {
		if ext == e {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	rel, err := filepath.Rel(h.appDir, path)
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

// scheduleRebuild resets the debounce timer to fire 150 ms from now.
func (h *HotReloader) scheduleRebuild() {
	h.timerMu.Lock()
	defer h.timerMu.Unlock()
	if h.timer != nil {
		h.timer.Reset(150 * time.Millisecond)
	} else {
		h.timer = time.AfterFunc(150*time.Millisecond, h.rebuild)
	}
}

// rebuild runs a full build and, on success, swaps in the new App and
// publishes a "reload" message so all connected browsers reload.
func (h *HotReloader) rebuild() {
	if !h.building.CompareAndSwap(false, true) {
		return
	}
	defer h.building.Store(false)

	fmt.Println("[hotreload] rebuilding...")
	newApp, err := Build(h.cfg)
	if err != nil {
		fmt.Printf("[hotreload] build error: %v\n", err)
		return
	}

	live := &liveApp{app: newApp, handler: newApp}
	h.current.Store(live)
	fmt.Println("[hotreload] rebuild complete")
	h.bus.Publish("update", []byte("reload"))
}

// ── WebSocket server ──────────────────────────────────────────────────────────

type wsServer struct {
	ps subscriber
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	sub := s.ps.Subscribe("update")
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

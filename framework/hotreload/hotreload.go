package hotreload

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"gojsx/framework"
	"gojsx/framework/pubsub"
)

// liveApp pairs a built App with its cached Handler.
type liveApp struct {
	app     *framework.App
	handler http.Handler
}

// HotReloader watches the app directory for source file changes, rebuilds,
// and notifies browsers via WebSocket using a pubsub bus.
type HotReloader struct {
	cfg       *framework.Config
	postBuild func(*framework.App)
	absAppDir string
	bus       *pubsub.Memory
	current   atomic.Pointer[liveApp]
	watcher   *fsnotify.Watcher
	timerMu   sync.Mutex
	timer     *time.Timer
	building  atomic.Bool
}

// ClientScript is the browser-side WebSocket listener that triggers a page
// reload when the dev server rebuilds. Assigned to framework.DevScript by New().
const ClientScript = `(function(){` +
	`function connect(){` +
	`var ws=new WebSocket("ws://"+location.host+"/__dev__/hot");` +
	`ws.onmessage=function(e){if(e.data==="reload")location.reload()};` +
	`ws.onclose=function(){setTimeout(connect,2000)};` +
	`}connect();` +
	`})();`

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// New creates a HotReloader watching cfg.AppDir for source changes.
// initial is the already-built App; postBuild (optional) is called after each
// successful rebuild to re-apply per-route overrides.
func New(cfg *framework.Config, initial *framework.App, postBuild func(*framework.App)) (*HotReloader, error) {
	framework.DevScript = ClientScript
	absAppDir, err := filepath.Abs(cfg.AppDir)
	if err != nil {
		return nil, fmt.Errorf("hotreload: abs app dir: %w", err)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("hotreload: watcher: %w", err)
	}

	h := &HotReloader{
		cfg:       cfg,
		postBuild: postBuild,
		absAppDir: absAppDir,
		bus:       pubsub.New(),
		watcher:   w,
	}

	live := &liveApp{app: initial, handler: initial.Handler()}
	h.current.Store(live)

	if err := watchDir(w, absAppDir); err != nil {
		w.Close()
		return nil, fmt.Errorf("hotreload: watch: %w", err)
	}

	go h.watchLoop()
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

// Close stops the file watcher.
func (h *HotReloader) Close() error {
	h.timerMu.Lock()
	if h.timer != nil {
		h.timer.Stop()
	}
	h.timerMu.Unlock()
	return h.watcher.Close()
}

// watchLoop receives fsnotify events and schedules debounced rebuilds.
func (h *HotReloader) watchLoop() {
	for {
		select {
		case event, ok := <-h.watcher.Events:
			if !ok {
				return
			}
			if !isSourceFile(h.absAppDir, event.Name) {
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
				h.scheduleRebuild()
			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("[hotreload] watcher error: %v\n", err)
		}
	}
}

// scheduleRebuild resets the debounce timer to fire 150ms from now.
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
	newApp, err := h.cfg.Build()
	if err != nil {
		fmt.Printf("[hotreload] build error: %v\n", err)
		return
	}

	if h.postBuild != nil {
		h.postBuild(newApp)
	}

	live := &liveApp{app: newApp, handler: newApp.Handler()}
	h.current.Store(live)
	fmt.Println("[hotreload] rebuild complete")
	h.bus.Publish("update", []byte("reload"))
}

// watchDir adds all source subdirectories of absDir to the watcher,
// skipping output and vendor directories.
func watchDir(w *fsnotify.Watcher, absDir string) error {
	return filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := filepath.Base(path)
			if name == "node_modules" || name == "public" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return w.Add(path)
		}
		return nil
	})
}

// isSourceFile returns true when path is a frontend source file that should
// trigger a rebuild.
func isSourceFile(absAppDir, path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx", ".ts", ".jsx", ".js", ".css":
	default:
		return false
	}
	rel, err := filepath.Rel(absAppDir, path)
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

// ── WebSocket server ──────────────────────────────────────────────────────────

type wsServer struct {
	ps pubsub.Subscriber
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

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

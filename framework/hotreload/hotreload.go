package hotreload

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"gojsx/framework"
	"gojsx/framework/pubsub"
)

// liveApp pairs a built App with its cached Handler.
type liveApp struct {
	app     *framework.App
	handler http.Handler
}

// HotReloader watches the app directory for source file changes, rebuilds,
// and notifies browsers via SSE using a pubsub bus.
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

// ClientScript is the browser-side SSE listener that triggers a page reload
// when the dev server rebuilds. Assigned to framework.DevScript by New().
const ClientScript = `(function(){var e=new EventSource("/__dev__/hot");` +
	`e.addEventListener("reload",function(){location.reload()});` +
	`e.addEventListener("error",function(ev){console.warn("[gojsx] build error:",ev.data)});` +
	`e.onerror=function(){setTimeout(function(){location.reload()},2000)};` +
	`})();`

// New creates a HotReloader watching cfg.AppDir for source changes.
// initial is the already-built App; postBuild (optional) is called after each
// successful rebuild to re-apply per-route overrides.
func New(cfg *framework.Config, initial *framework.App, postBuild func(*framework.App)) (*HotReloader, error) {
	framework.DevScript = ClientScript
	// Resolve to absolute so filepath.Rel works correctly against
	// the absolute paths fsnotify reports on macOS (FSEvents).
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

// Handler returns an http.Handler that serves SSE at /__dev__/hot
// and delegates all other requests to the current live App.
func (h *HotReloader) Handler() http.Handler {
	sseServer := &sseServer{ps: h.bus}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__dev__/hot" {
			sseServer.ServeHTTP(w, r)
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
// publishes an "update" event so all connected browsers reload.
func (h *HotReloader) rebuild() {
	if !h.building.CompareAndSwap(false, true) {
		return
	}
	defer h.building.Store(false)

	fmt.Println("[hotreload] rebuilding...")
	newApp, err := h.cfg.Build()
	if err != nil {
		fmt.Printf("[hotreload] build error: %v\n", err)
		h.bus.Publish("error", []byte(err.Error()))
		return
	}

	if h.postBuild != nil {
		h.postBuild(newApp)
	}

	live := &liveApp{app: newApp, handler: newApp.Handler()}
	h.current.Store(live)
	fmt.Println("[hotreload] rebuild complete")
	h.bus.Publish("update", []byte(`{"reload":true}`))
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
// trigger a rebuild. Uses absolute paths so filepath.Rel is reliable on macOS
// where fsnotify (FSEvents) reports absolute paths regardless of how dirs were added.
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

// ── SSE server ────────────────────────────────────────────────────────────────

type sseServer struct {
	ps pubsub.Subscriber
}

func (s *sseServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "hot: response writer is not a flusher", http.StatusInternalServerError)
		return
	}
	headers := w.Header()
	headers.Add("Content-Type", "text/event-stream")
	headers.Add("Cache-Control", "no-cache")
	headers.Add("Connection", "keep-alive")
	headers.Add("Access-Control-Allow-Origin", "*")
	flusher.Flush()

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
			w.Write(sseEvent{Type: "reload", Data: data}.format().Bytes()) //nolint:errcheck
			flusher.Flush()
		}
	}
}

// sseEvent is a server-sent event.
// https://html.spec.whatwg.org/multipage/server-sent-events.html
type sseEvent struct {
	ID    string
	Type  string
	Data  []byte
	Retry int
}

func (e sseEvent) format() *bytes.Buffer {
	b := new(bytes.Buffer)
	if e.ID != "" {
		fmt.Fprintf(b, "id: %s\n", e.ID)
	}
	if e.Type != "" {
		fmt.Fprintf(b, "event: %s\n", e.Type)
	}
	if len(e.Data) > 0 {
		b.WriteString("data: ")
		b.Write(e.Data)
		b.WriteByte('\n')
	}
	if e.Retry > 0 {
		fmt.Fprintf(b, "retry: %s\n", strconv.Itoa(e.Retry))
	}
	b.WriteByte('\n')
	return b
}

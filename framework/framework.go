package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gojsx/framework/contract"
)

// ── Config ───────────────────────────────────────────────────────────────────

// Config is the top-level wiring configuration for a GoJSX application.
// Every interface field has a built-in default (React + esbuild + Goja).
// Set only the fields you want to override.
type Config struct {
	// Source location
	AppDir    string
	PublicDir string
	Dev       bool

	// ClientEntry is the browser bootstrap — a file path or a Node import
	// specifier (e.g. "@gojsx/react/client"). If empty, the framework
	// looks for AppDir/_client.tsx; if absent, it uses the registered default.
	ClientEntry string

	// Global Go→JS bridge (all routes fall back to this unless overridden per-route)
	GlobalBridge contract.BridgeConfig

	// Build pipeline (nil = built-in defaults)
	Discoverer     Discoverer
	Bundler        Bundler
	EntryGenerator ServerEntryGenerator
	RouteBuilder   RouteBuilder

	// Runtime (nil = built-in Goja defaults)
	StreamProtocol StreamProtocol

	// HTTP layer (nil = built-in defaults)
	HTMLShell   HTMLShell
	AssetServer AssetServer
}

// ── BuildArtifacts ───────────────────────────────────────────────────────────

// BuildArtifacts is the output of the build pipeline stage.
type BuildArtifacts struct {
	Output               contract.BundleOutput
	Routes               []contract.Route
	GlobalNotFoundExport string // "GlobalNotFound" if global-not-found.tsx exists, else ""
}

// ── App ──────────────────────────────────────────────────────────────────────

// App is the fully-wired application returned by Config.Build.
// It implements http.Handler via Handler().
type App struct {
	cfg       Config
	artifacts BuildArtifacts
	pool      VMPool
	renderer  Renderer
	protocol  StreamProtocol
	router    Router
	shell     HTMLShell
	assets    AssetServer

	// SSRStreaming mirrors the server package's SSR_STREAMING env flag.
	streaming bool
}

// Build wires all components, runs the build pipeline, and returns a ready
// App or an error. Nil Config fields are replaced with built-in defaults.
func (c *Config) Build() (*App, error) {
	if err := c.fillDefaults(); err != nil {
		return nil, err
	}

	// ── 1. Discover ───────────────────────────────────────────────────────
	disc, err := c.Discoverer.Discover(c.AppDir)
	if err != nil {
		return nil, fmt.Errorf("framework: discover: %w", err)
	}

	// ── 2. Bundle ─────────────────────────────────────────────────────────
	pubURLName := "public"
	if c.PublicDir != "" {
		pubURLName = filepath.Base(strings.TrimSuffix(c.PublicDir, "/"))
	}
	assetsURLPath := "/" + pubURLName + "/assets"

	entryContent, err := c.EntryGenerator.Generate(EntryGenConfig{
		AppDir:             c.AppDir,
		Pages:              disc.Pages,
		GlobalNotFoundPath: disc.GlobalNotFound,
		GlobalErrorPath:    disc.GlobalError,
	})
	if err != nil {
		return nil, fmt.Errorf("framework: entry generate: %w", err)
	}

	output, err := c.Bundler.Bundle(contract.BundleInput{
		AppDir:                 c.AppDir,
		OutDir:                 c.PublicDir + "/assets",
		AssetsURLPath:          assetsURLPath,
		ClientEntry:            c.ClientEntry,
		ClientComponents:       disc.ClientComponents,
		Dev:                    c.Dev,
		ServerEntryContent:     entryContent,
		ServerBundleConditions: c.EntryGenerator.BundleConditions(),
	})
	if err != nil {
		return nil, fmt.Errorf("framework: bundle: %w", err)
	}

	// ── 3. Boot VM factory & pool ─────────────────────────────────────────
	factory, err := c.newVMFactory(output.ServerBundle)
	if err != nil {
		return nil, fmt.Errorf("framework: vm factory: %w", err)
	}
	pool, err := newFactoryPool(factory, c.GlobalBridge)
	if err != nil {
		return nil, fmt.Errorf("framework: vm pool: %w", err)
	}

	// ── 4. Build routes ───────────────────────────────────────────────────
	router := &priorityRouter{}
	globalNotFoundExport := ""
	for _, p := range disc.Pages {
		route := c.RouteBuilder.Build(c.AppDir, p)
		router.register(route)
	}
	if disc.GlobalNotFound != "" {
		globalNotFoundExport = "GlobalNotFound"
	}

	// ── 5. Renderer ───────────────────────────────────────────────────────
	renderer := newFrameworkRenderer(pool, c.StreamProtocol, c.GlobalBridge)

	// ── 6. Wire HTTP layer ────────────────────────────────────────────────
	app := &App{
		cfg: *c,
		artifacts: BuildArtifacts{
			Output:               output,
			Routes:               router.routes,
			GlobalNotFoundExport: globalNotFoundExport,
		},
		pool:      pool,
		renderer:  renderer,
		protocol:  c.StreamProtocol,
		router:    router,
		shell:     c.HTMLShell,
		assets:    c.AssetServer,
		streaming: os.Getenv("SSR_STREAMING") != "false",
	}
	return app, nil
}

// Artifacts returns the build artifacts for inspection or per-route override.
func (a *App) Artifacts() *BuildArtifacts { return &a.artifacts }

// Handler returns an http.Handler for the entire application.
// Static assets are served under PublicDir; all other paths go to ServeHTTP.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	pubURLName := "public"
	if a.cfg.PublicDir != "" {
		pubURLName = filepath.Base(strings.TrimSuffix(a.cfg.PublicDir, "/"))
	}
	publicPrefix := "/" + pubURLName + "/"
	mux.Handle(publicPrefix, a.assets.Handler(publicPrefix))
	mux.HandleFunc("/", a.ServeHTTP)
	return mux
}

// ServeHTTP is the low-level http.HandlerFunc for all page routes.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, params := a.router.Match(r.URL.Path)

	htmlParams := contract.ShellParams{
		ImportURLs:   a.artifacts.Output.ImportURLs,
		ClientScript: a.artifacts.Output.ClientEntryURL,
	}

	if route == nil {
		if a.artifacts.GlobalNotFoundExport == "" {
			http.NotFound(w, r)
			return
		}
		if a.protocol.IsStreamingRequest(r) {
			w.Header().Set("Content-Type", a.protocol.ContentType()+"; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			sw := newHTTPStreamWriter(w)
			_ = a.renderer.Render(r.Context(), sw, contract.RenderOpts{
				ExportName: a.artifacts.GlobalNotFoundExport,
				Props:      map[string]any{},
			})
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, a.shell.Render(htmlParams))
		}
		return
	}

	props := buildPageProps(r, params)

	if a.protocol.IsStreamingRequest(r) {
		w.Header().Set("Content-Type", a.protocol.ContentType()+"; charset=utf-8")
		sw := newHTTPStreamWriter(w)
		if err := a.renderer.Render(r.Context(), sw, contract.RenderOpts{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: requestCtx(r),
			Bridge:         route.Bridge,
		}); err != nil {
			fmt.Printf("rsc %s: %v\n", r.URL.Path, err)
			fmt.Fprintf(w, `<div class="rsc-err">%s</div>`, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if !a.streaming {
		var buf bytes.Buffer
		bsw := &bufStreamWriter{buf: &buf}
		if err := a.renderer.Render(r.Context(), bsw, contract.RenderOpts{
			ExportName:     route.Export,
			Props:          props,
			RequestContext: requestCtx(r),
			Bridge:         route.Bridge,
		}); err != nil {
			fmt.Printf("html render %s: %v\n", r.URL.Path, err)
		}
		flightJSON, _ := json.Marshal(buf.String())
		htmlParams.Scripts = append(htmlParams.Scripts, "self.__flight_data="+string(flightJSON))
	}

	fmt.Fprint(w, a.shell.Render(htmlParams))
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// fillDefaults installs built-in implementations for any nil Config fields.
// Returns an error if AppDir is empty.
func (c *Config) fillDefaults() error {
	if c.AppDir == "" {
		return fmt.Errorf("framework: Config.AppDir must be set")
	}
	if c.PublicDir == "" {
		c.PublicDir = filepath.Join(c.AppDir, "public")
	}
	if c.ClientEntry == "" {
		if _, err := os.Stat(filepath.Join(c.AppDir, "_client.tsx")); err == nil {
			c.ClientEntry = filepath.Join(c.AppDir, "_client.tsx")
		} else {
			c.ClientEntry = defaultClientEntry
		}
	}

	// We resolve these lazily via import paths to avoid import cycles.
	// The concrete types are in the build, html, server, and runtime packages.
	// Config.fillDefaults uses function variables (set by each package's init)
	// for the defaults.
	if c.Discoverer == nil {
		if defaultDiscoverer == nil {
			return fmt.Errorf("framework: no default Discoverer registered; import gojsx/build")
		}
		c.Discoverer = defaultDiscoverer()
	}
	if c.Bundler == nil {
		if defaultBundler == nil {
			return fmt.Errorf("framework: no default Bundler registered; import gojsx/build")
		}
		c.Bundler = defaultBundler()
	}
	if c.EntryGenerator == nil {
		if defaultEntryGenerator == nil {
			return fmt.Errorf("framework: no default EntryGenerator registered; import gojsx/build")
		}
		c.EntryGenerator = defaultEntryGenerator()
	}
	if c.RouteBuilder == nil {
		if defaultRouteBuilder == nil {
			return fmt.Errorf("framework: no default RouteBuilder registered; import gojsx/build")
		}
		c.RouteBuilder = defaultRouteBuilder()
	}
	if c.StreamProtocol == nil {
		if defaultStreamProtocol == nil {
			return fmt.Errorf("framework: no default StreamProtocol registered; import gojsx/runtime")
		}
		c.StreamProtocol = defaultStreamProtocol()
	}
	if c.HTMLShell == nil {
		if defaultHTMLShell == nil {
			return fmt.Errorf("framework: no default HTMLShell registered; import gojsx/html")
		}
		c.HTMLShell = defaultHTMLShell()
	}
	if c.AssetServer == nil {
		if defaultAssetServer == nil {
			return fmt.Errorf("framework: no default AssetServer registered; import gojsx/server")
		}
		c.AssetServer = defaultAssetServer(c.PublicDir)
	}
	return nil
}

// newVMFactory creates a VMFactory from the registered implementation.
// Uses a function variable so framework.go doesn't directly import vm/goja.
func (c *Config) newVMFactory(serverBundle []byte) (VMFactory, error) {
	if defaultVMFactory == nil {
		return nil, fmt.Errorf("framework: no default VMFactory registered; import gojsx/vm/goja")
	}
	return defaultVMFactory(serverBundle)
}

// ── Default registry ─────────────────────────────────────────────────────────
// Concrete packages register their constructors here via init() functions,
// avoiding direct imports from framework into implementation packages.

var (
	defaultDiscoverer      func() Discoverer
	defaultBundler         func() Bundler
	defaultEntryGenerator  func() ServerEntryGenerator
	defaultRouteBuilder    func() RouteBuilder
	defaultStreamProtocol  func() StreamProtocol
	defaultHTMLShell       func() HTMLShell
	defaultAssetServer     func(dir string) AssetServer
	defaultVMFactory       func(bundle []byte) (VMFactory, error)
	defaultRendererFactory func(pool VMPool, protocol StreamProtocol, bridge contract.BridgeConfig) Renderer
	defaultClientEntry     string
)

// RegisterDefaults is called by concrete packages (build, runtime, html, server)
// to register their default implementations. It is not intended for end-user code.
func RegisterDefaults(d Defaults) {
	if d.Discoverer != nil {
		defaultDiscoverer = d.Discoverer
	}
	if d.Bundler != nil {
		defaultBundler = d.Bundler
	}
	if d.EntryGenerator != nil {
		defaultEntryGenerator = d.EntryGenerator
	}
	if d.RouteBuilder != nil {
		defaultRouteBuilder = d.RouteBuilder
	}
	if d.StreamProtocol != nil {
		defaultStreamProtocol = d.StreamProtocol
	}
	if d.HTMLShell != nil {
		defaultHTMLShell = d.HTMLShell
	}
	if d.AssetServer != nil {
		defaultAssetServer = d.AssetServer
	}
	if d.VMFactory != nil {
		defaultVMFactory = d.VMFactory
	}
	if d.RendererFactory != nil {
		defaultRendererFactory = d.RendererFactory
	}
	if d.ClientEntry != "" {
		defaultClientEntry = d.ClientEntry
	}
}

// Defaults is passed to RegisterDefaults by concrete packages.
type Defaults struct {
	Discoverer      func() Discoverer
	Bundler         func() Bundler
	EntryGenerator  func() ServerEntryGenerator
	RouteBuilder    func() RouteBuilder
	StreamProtocol  func() StreamProtocol
	HTMLShell       func() HTMLShell
	AssetServer     func(dir string) AssetServer
	VMFactory       func(bundle []byte) (VMFactory, error)
	RendererFactory func(pool VMPool, protocol StreamProtocol, bridge contract.BridgeConfig) Renderer
	ClientEntry     string
}

// ── Internal Renderer ────────────────────────────────────────────────────────

func newFrameworkRenderer(pool VMPool, protocol StreamProtocol, bridge contract.BridgeConfig) Renderer {
	if defaultRendererFactory != nil {
		return defaultRendererFactory(pool, protocol, bridge)
	}
	return &inlineRenderer{pool: pool, protocol: protocol, bridge: bridge}
}

type inlineRenderer struct {
	pool     VMPool
	protocol StreamProtocol
	bridge   contract.BridgeConfig
}

func (r *inlineRenderer) Render(_ context.Context, w StreamWriter, opts contract.RenderOpts) error { //nolint:staticcheck
	vm := r.pool.Acquire()
	defer r.pool.Release(vm)

	if err := vm.SetRequestContext(opts.RequestContext); err != nil {
		return fmt.Errorf("renderer: set context: %w", err)
	}
	jsi := r.bridge.Context
	if opts.Bridge != nil {
		jsi = opts.Bridge.Context
	}
	if err := vm.SetBridgeFunctions(jsi); err != nil {
		return fmt.Errorf("renderer: set bridge: %w", err)
	}
	propsJSON, err := json.Marshal(opts.Props)
	if err != nil {
		return fmt.Errorf("renderer: marshal props: %w", err)
	}
	handle, err := vm.CallRenderFunction(opts.ExportName, string(propsJSON))
	if err != nil {
		return fmt.Errorf("renderer: call render: %w", err)
	}
	return r.protocol.Drain(vm, handle, w)
}

// ── factoryPool ───────────────────────────────────────────────────────────────

type factoryPool struct {
	factory VMFactory
	bridge  contract.BridgeConfig
	pool    sync.Pool
}

func newFactoryPool(factory VMFactory, bridge contract.BridgeConfig) (*factoryPool, error) {
	p := &factoryPool{factory: factory, bridge: bridge}
	p.pool = sync.Pool{
		New: func() any {
			vm, err := factory.New(bridge)
			if err != nil {
				panic(fmt.Sprintf("framework: pool create: %v", err))
			}
			return vm
		},
	}
	// Eagerly create one VM to catch startup errors immediately.
	vm := p.pool.New()
	p.pool.Put(vm)
	return p, nil
}

func (p *factoryPool) Acquire() VM { return p.pool.Get().(VM) }

func (p *factoryPool) Release(vm VM) {
	if err := vm.ClearState(); err == nil {
		p.pool.Put(vm)
	}
}

// ── priorityRouter ───────────────────────────────────────────────────────────

type priorityRouter struct {
	routes   []contract.Route
	sortOnce sync.Once
}

func (r *priorityRouter) Register(route contract.Route) { r.routes = append(r.routes, route) }
func (r *priorityRouter) register(route contract.Route) { r.Register(route) }

func (r *priorityRouter) Match(path string) (*contract.Route, map[string]any) {
	r.sortOnce.Do(func() {
		sortRoutes(r.routes)
	})
	for i := range r.routes {
		if params, ok := MatchPattern(r.routes[i].Pattern, path); ok {
			return &r.routes[i], params
		}
	}
	return nil, nil
}

// MatchPattern matches a URL path against a route pattern.
// Supports static segments, dynamic segments (":name"), catch-all (":...name"),
// and optional catch-all (":...name?").
func MatchPattern(pattern, path string) (map[string]any, bool) {
	pp := splitPath(pattern)
	rp := splitPath(path)
	params := map[string]any{}
	for i, seg := range pp {
		if strings.HasPrefix(seg, ":...") {
			optional := strings.HasSuffix(seg, "?")
			name := seg[4:]
			if optional {
				name = name[:len(name)-1]
			}
			remaining := rp[i:]
			if len(remaining) == 0 || (len(remaining) == 1 && remaining[0] == "") {
				if !optional {
					return nil, false
				}
				return params, true
			}
			params[name] = remaining
			return params, true
		}
		if i >= len(rp) {
			return nil, false
		}
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = rp[i]
		} else if seg != rp[i] {
			return nil, false
		}
	}
	if len(pp) != len(rp) {
		return nil, false
	}
	return params, true
}

func splitPath(p string) []string { return strings.Split(p, "/") }

func routeScoreInternal(pattern string) int {
	score := 0
	for _, seg := range splitPath(pattern) {
		switch {
		case strings.HasPrefix(seg, ":...") && strings.HasSuffix(seg, "?"):
			score -= 20
		case strings.HasPrefix(seg, ":..."):
			score -= 10
		case strings.HasPrefix(seg, ":"):
			// neutral
		default:
			score++
		}
	}
	return score
}

func sortRoutes(routes []contract.Route) {
	n := len(routes)
	for i := 1; i < n; i++ {
		for j := i; j > 0 && routeScoreInternal(routes[j-1].Pattern) < routeScoreInternal(routes[j].Pattern); j-- {
			routes[j-1], routes[j] = routes[j], routes[j-1]
		}
	}
}

// ── HTTP helpers ─────────────────────────────────────────────────────────────

type httpStreamWriter struct {
	w       io.Writer
	flusher http.Flusher
}

func newHTTPStreamWriter(w http.ResponseWriter) *httpStreamWriter {
	sw := &httpStreamWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		sw.flusher = f
	}
	return sw
}

func (sw *httpStreamWriter) WriteRaw(p []byte) (int, error) { return sw.w.Write(p) }
func (sw *httpStreamWriter) Flush() {
	if sw.flusher != nil {
		sw.flusher.Flush()
	}
}

type bufStreamWriter struct {
	buf *bytes.Buffer
}

func (bsw *bufStreamWriter) WriteRaw(p []byte) (int, error) { return bsw.buf.Write(p) }
func (bsw *bufStreamWriter) Flush()                         {}

func buildPageProps(r *http.Request, params map[string]any) map[string]any {
	searchParams := map[string]any{}
	for k, vs := range r.URL.Query() {
		if len(vs) == 1 {
			searchParams[k] = vs[0]
		} else {
			searchParams[k] = vs
		}
	}
	return map[string]any{"params": params, "searchParams": searchParams}
}

func requestCtx(r *http.Request) map[string]any {
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}
	return map[string]any{
		"url":     r.URL.String(),
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"method":  r.Method,
		"headers": headers,
	}
}

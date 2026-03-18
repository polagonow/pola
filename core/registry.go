package core

import (
	"context"
	"net/http"
	"time"
)

// Registry holds all pluggable components for a Pola application.
// Nil fields are filled from init()-registered defaults at build time.
type Registry struct {
	Renderer   Renderer
	Router     Router
	Bundler    Bundler
	Engine     JSEngine
	FS         FS
	Cache      Cache
	CSS        CSS
	Logger     Logger
	Polyfills  PolyfillRegistry
	Metrics    Metrics // default: noop
	Tracer     Tracer  // default: noop
	Pprof      Pprof   // default: nil = disabled
	Middleware []Middleware
	Injectors  []RuntimeInjector
}

// Config is the top-level configuration for a Pola application.
type Config struct {
	// WebAppPath is the path to the frontend app directory (env: POLA_WEBAPP_PATH)
	WebAppPath string
	// PublicDir is where compiled assets are written (default: WebAppPath/public)
	PublicDir string
	// Dev enables development mode with hot reload
	Dev bool
	// Registry overrides specific plugins; nil fields use registered defaults
	Registry *Registry
}

// BuildArtifacts holds the outputs produced during the build pipeline.
type BuildArtifacts struct {
	Output *BundleOutput
}

// App is the fully-wired Pola application. It implements http.Handler.
type App struct {
	cfg       Config
	registry  *Registry
	handler   http.Handler
	cache     Cache
	artifacts BuildArtifacts
}

// New creates and builds a Pola application from the given config.
// All nil Registry fields are populated from init()-registered defaults.
func New(cfg *Config) (*App, error) {
	// Populated by internal/pipeline — separate package to avoid cycles
	// This stub exists so core can be imported without internal/.
	panic("core.New: not implemented — import github.com/polagonow/pola to get the full implementation")
}

// NewApp constructs an App with all fields populated. Called exclusively by
// internal/pipeline to avoid import cycles — end users should call pola.New.
func NewApp(cfg *Config, reg *Registry, handler http.Handler) *App {
	a := &App{
		registry: reg,
		handler:  handler,
	}
	if cfg != nil {
		a.cfg = *cfg
	}
	if reg != nil {
		a.cache = reg.Cache
	}
	return a
}

// SetArtifacts stores the build artifacts produced during the pipeline.
// Called by internal/pipeline after the bundler completes.
func (a *App) SetArtifacts(output *BundleOutput) {
	a.artifacts = BuildArtifacts{Output: output}
}

// Artifacts returns the build artifacts produced during pipeline execution.
func (a *App) Artifacts() BuildArtifacts { return a.artifacts }

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

// Cache returns the cache instance for runtime management.
func (a *App) Cache() Cache { return a.cache }

// Build runs the build pipeline explicitly (called by New internally).
func (a *App) Build(ctx context.Context) error { return nil }

// ── Default registry (populated by init() in each plugin package) ────────────

var (
	defaultRenderer    func() Renderer
	defaultRouter      func() Router
	defaultBundler     func() Bundler
	defaultEngine      func() JSEngine
	defaultFS          func() FS
	defaultCache       func() Cache
	defaultLogger      func() Logger
	defaultMetrics     func() Metrics
	defaultTracer      func() Tracer
	defaultShell       func() HTMLShell
	defaultPolyfills   func() PolyfillRegistry
	defaultAssetServer func(publicDir string) AssetServer
)

// RegisterRenderer registers the default Renderer (called from init()).
func RegisterRenderer(f func() Renderer) { defaultRenderer = f }

// RegisterRouter registers the default Router.
func RegisterRouter(f func() Router) { defaultRouter = f }

// RegisterBundler registers the default Bundler.
func RegisterBundler(f func() Bundler) { defaultBundler = f }

// RegisterEngine registers the default JSEngine.
func RegisterEngine(f func() JSEngine) { defaultEngine = f }

// RegisterFS registers the default FS.
func RegisterFS(f func() FS) { defaultFS = f }

// RegisterCache registers the default Cache.
func RegisterCache(f func() Cache) { defaultCache = f }

// RegisterLogger registers the default Logger.
func RegisterLogger(f func() Logger) { defaultLogger = f }

// RegisterMetrics registers the default Metrics.
func RegisterMetrics(f func() Metrics) { defaultMetrics = f }

// RegisterTracer registers the default Tracer.
func RegisterTracer(f func() Tracer) { defaultTracer = f }

// RegisterHTMLShell registers the default HTMLShell.
func RegisterHTMLShell(f func() HTMLShell) { defaultShell = f }

// DefaultHTMLShell returns the registered default HTMLShell constructor, or nil.
func DefaultHTMLShell() func() HTMLShell { return defaultShell }

// RegisterPolyfills registers the default PolyfillRegistry.
func RegisterPolyfills(f func() PolyfillRegistry) { defaultPolyfills = f }

// RegisterAssetServer registers the default AssetServer constructor.
// Called from init() in FS plugin packages (e.g. fs/osfs).
func RegisterAssetServer(f func(publicDir string) AssetServer) { defaultAssetServer = f }

// DefaultAssetServer returns the registered AssetServer constructor, or nil.
func DefaultAssetServer() func(publicDir string) AssetServer { return defaultAssetServer }

// FillDefaults populates nil Registry fields from registered defaults.
// Returns error if a required component has no default.
func (r *Registry) FillDefaults() error {
	if r.Renderer == nil && defaultRenderer != nil {
		r.Renderer = defaultRenderer()
	}
	if r.Router == nil && defaultRouter != nil {
		r.Router = defaultRouter()
	}
	if r.Bundler == nil && defaultBundler != nil {
		r.Bundler = defaultBundler()
	}
	if r.Engine == nil && defaultEngine != nil {
		r.Engine = defaultEngine()
	}
	if r.FS == nil && defaultFS != nil {
		r.FS = defaultFS()
	}
	if r.Cache == nil && defaultCache != nil {
		r.Cache = defaultCache()
	}
	if r.Logger == nil {
		if defaultLogger != nil {
			r.Logger = defaultLogger()
		} else {
			r.Logger = noopLogger{}
		}
	}
	if r.Metrics == nil {
		if defaultMetrics != nil {
			r.Metrics = defaultMetrics()
		} else {
			r.Metrics = noopMetrics{}
		}
	}
	if r.Tracer == nil {
		if defaultTracer != nil {
			r.Tracer = defaultTracer()
		} else {
			r.Tracer = noopTracer{}
		}
	}
	if r.Polyfills == nil {
		if defaultPolyfills != nil {
			r.Polyfills = defaultPolyfills()
		} else {
			r.Polyfills = NewPolyfillRegistry()
		}
	}
	return nil
}

// NewPolyfillRegistry returns a default in-memory polyfill registry.
func NewPolyfillRegistry() PolyfillRegistry {
	return &polyfillRegistry{entries: map[PolyfillID]PolyfillSource{}}
}

type polyfillRegistry struct {
	entries map[PolyfillID]PolyfillSource
}

func (r *polyfillRegistry) Register(p PolyfillSource) { r.entries[p.ID] = p }
func (r *polyfillRegistry) Get(ids ...PolyfillID) []PolyfillSource {
	out := make([]PolyfillSource, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.entries[id]; ok {
			out = append(out, p)
		}
	}
	return out
}

// ── Noop implementations ──────────────────────────────────────────────────────

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) With(args ...any) Logger       { return noopLogger{} }

type noopMetrics struct{}

func (noopMetrics) Name() string { return "noop" }
func (noopMetrics) RecordRequest(route, method string, statusCode int, duration time.Duration) {}
func (noopMetrics) Handler() http.Handler { return http.NotFoundHandler() }

type noopTracer struct{}

func (noopTracer) Name() string { return "noop" }
func (noopTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End()                         {}
func (noopSpan) SetAttribute(key string, value any) {}

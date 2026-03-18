package core

import (
	"context"
	"net/http"
	"time"
)

// PolyfillID identifies a polyfill by name.
type PolyfillID string

// PolyfillSource is a named polyfill JS source.
type PolyfillSource struct {
	ID     PolyfillID
	Source string
}

// PolyfillRegistry manages available polyfills.
type PolyfillRegistry interface {
	Register(p PolyfillSource)
	Get(ids ...PolyfillID) []PolyfillSource
}

// Capability describes a renderer's capabilities.
type Capability string

// JSRuntime is a single JS execution context.
type JSRuntime interface {
	Eval(script string) (any, error)
	Call(fn string, args ...any) (any, error)
	Set(name string, value any) error
	Dispose()
}

// JSEngine creates JS runtimes.
type JSEngine interface {
	Name() string
	NewRuntime(ctx context.Context) (JSRuntime, error)
	RequiredPolyfills() []PolyfillID
}

// Renderer renders a page request.
type Renderer interface {
	Name() string
	FileExtensions() []string
	Render(ctx context.Context, req RenderRequest) (RenderResult, error)
	Capabilities() []Capability
}

// Router scans for routes and resolves requests.
type Router interface {
	Name() string
	ScanRoutes(ctx context.Context, fs FS, appDir string, exts []string) ([]Route, error)
	Resolve(ctx context.Context, path string) (*Route, map[string]any)
}

// Bundler builds JS bundles.
type Bundler interface {
	Name() string
	Build(ctx context.Context, req BundleInput) (*BundleOutput, error)
	Watch(ctx context.Context, req BundleInput, onChange func(*BundleOutput)) error
}

// FS is the file system abstraction.
type FS interface {
	Name() string
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]FSFileInfo, error)
	Exists(path string) bool
	Watch(path string, onChange func(string)) error
}

// Cache stores rendered output.
type Cache interface {
	Name() string
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, opts CacheOptions) error
	Delete(ctx context.Context, key string) error
	Invalidate(ctx context.Context, prefix string) error
	Clear(ctx context.Context) error
}

// RuntimeInjector injects Go services into the JS runtime as __DEPENDENCY_INJECTION__ functions.
type RuntimeInjector interface {
	Name() string
	Inject(ctx context.Context, runtime JSRuntime) error
	Capabilities() []InjectionCapability
}

// CSS processes CSS files.
type CSS interface {
	Name() string
	Process(ctx context.Context, inputPath string, outputPath string) error
	Watch(ctx context.Context, inputPath string, outputPath string, onChange func()) error
}

// Middleware wraps HTTP handlers.
type Middleware interface {
	Name() string
	Wrap(next http.Handler) http.Handler
}

// Logger is the framework logger.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	With(args ...any) Logger
}

// Span is an OpenTelemetry-compatible trace span.
type Span interface {
	End()
	SetAttribute(key string, value any)
}

// Metrics records observability metrics.
type Metrics interface {
	Name() string
	Path() string // HTTP path where the metrics endpoint is served (e.g. "/metrics")
	RecordRequest(route, method string, statusCode int, duration time.Duration)
	RecordRender(route string, duration time.Duration)
	Handler() http.Handler
}

// Tracer creates trace spans.
type Tracer interface {
	Name() string
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Pprof serves profiling endpoints.
type Pprof interface {
	Name() string
	Path() string // HTTP path prefix where pprof endpoints are served (e.g. "/debug/pprof")
	Handler() http.Handler
}

// LogAware is an optional interface implemented by components that accept a Logger.
// The pipeline calls SetLogger on all registered components after FillDefaults.
type LogAware interface {
	SetLogger(Logger)
}

// HotReloader watches for file changes and triggers rebuilds.
type HotReloader interface {
	Watch(ctx context.Context) error
	NotifyBrowser()
	Close() error
}

// StreamWriter is the destination for streaming render output.
type StreamWriter interface {
	WriteRaw(p []byte) (int, error)
	Flush()
}

// StreamHandle is an opaque reference to an in-progress rendering stream.
// It is returned by SSRRuntime.CallRenderFunction and consumed by DrainStream.
type StreamHandle interface {
	IsNil() bool
}

// SSRRuntime extends JSRuntime with methods for streaming SSR rendering
// (RSC Flight protocol). Implemented by JS runtimes that support React SSR.
type SSRRuntime interface {
	JSRuntime
	// SetRequestContext injects per-request data as __REQUEST__ in the runtime.
	SetRequestContext(ctx map[string]any) error
	// CallRenderFunction calls the server-side render entry point and returns
	// an opaque handle to the resulting ReadableStream.
	CallRenderFunction(exportName, propsJSON string) (StreamHandle, error)
	// DrainStream polls the stream until done, writing decoded chunks to w.
	// Returns (true, nil) if at least one byte was written.
	DrainStream(handle StreamHandle, w StreamWriter) (bool, error)
}

// SSRPool manages a pool of SSRRuntime instances backed by a compiled bundle.
type SSRPool interface {
	Acquire() SSRRuntime
	Release(rt SSRRuntime)
}

// SSRPoolFactory is implemented by JS engines that can create an SSR VM pool
// from a compiled server bundle. The pipeline calls NewSSRPool after bundling.
type SSRPoolFactory interface {
	NewSSRPool(bundle []byte) (SSRPool, error)
}

// BundleLoader is implemented by renderers that need the compiled server bundle
// to initialize their JS runtime pool. The pipeline calls LoadBundle immediately
// after the bundler produces output.
type BundleLoader interface {
	LoadBundle(engine JSEngine, bundle []byte) error
}

// HTMLShell renders the initial HTML document.
type HTMLShell interface {
	Render(p ShellParams) string
}

// AssetServer serves static assets.
type AssetServer interface {
	Handler(prefix string) http.Handler
}

package core

import (
	"context"
	"net/http"

	"github.com/samber/do/v2"
)

// Config holds resolved configuration for a Pola application.
// Constructed internally from Option functions — not created directly by users.
type Config struct {
	WebAppPath string
	PublicDir  string
	HTTPPath   string // URL path prefix; defaults to "/"
	Dev        bool
	APIOnly    bool
}

// Option configures a Pola application.
type Option func(*Config)

// WithWebAppDir sets the path to the frontend app directory.
func WithWebAppDir(path string) Option {
	return func(c *Config) { c.WebAppPath = path }
}

// WithPublicDir sets where compiled assets are written.
func WithPublicDir(dir string) Option {
	return func(c *Config) { c.PublicDir = dir }
}

// WithHTTPPath sets the URL path prefix the app is served under (default "/").
func WithHTTPPath(path string) Option {
	return func(c *Config) { c.HTTPPath = path }
}

// WithDev enables development mode with hot reload.
func WithDev(dev bool) Option {
	return func(c *Config) { c.Dev = dev }
}

// WithAPIOnly configures the app as API-only (no frontend/renderer).
func WithAPIOnly(v bool) Option {
	return func(c *Config) { c.APIOnly = v }
}

// Registry is the explicit service registration API.
// It wraps do.Injector internally so existing resolution code keeps working.
type Registry struct {
	injector   do.Injector
	middleware []Middleware
	injectors  []RuntimeInjector
}

// NewRegistry creates a new empty Registry backed by a do.Injector.
func NewRegistry() *Registry {
	return &Registry{
		injector: do.New(),
	}
}

// Provide registers a singleton service by its concrete type.
func Provide[T any](r *Registry, fn func() (T, error)) {
	do.Provide(r.injector, func(_ do.Injector) (T, error) {
		return fn()
	})
}

// ProvideValue registers a pre-constructed singleton value.
func ProvideValue[T any](r *Registry, val T) {
	do.ProvideValue(r.injector, val)
}

// Invoke resolves a service by type from the registry.
func Invoke[T any](r *Registry) (T, error) {
	return do.Invoke[T](r.injector)
}

// MustInvoke resolves a service by type, panicking on error.
func MustInvoke[T any](r *Registry) T {
	return do.MustInvoke[T](r.injector)
}

// AddMiddleware appends a middleware to the registry.
func (r *Registry) AddMiddleware(m Middleware) {
	r.middleware = append(r.middleware, m)
}

// Middleware returns all registered middleware in registration order.
func (r *Registry) Middleware() []Middleware { return r.middleware }

// AddInjector appends a RuntimeInjector to the registry.
func (r *Registry) AddInjector(inj RuntimeInjector) {
	r.injectors = append(r.injectors, inj)
}

// Injectors returns all registered RuntimeInjectors in registration order.
func (r *Registry) RuntimeInjectors() []RuntimeInjector { return r.injectors }

// BuildArtifacts holds the outputs produced during the build pipeline.
type BuildArtifacts struct {
	Output *BundleOutput
}

// PrebuildArtifacts holds all data pre-computed during mage Bundle so that the
// production binary (embed tag) can skip route scanning and JS bundling entirely.
type PrebuildArtifacts struct {
	Routes         []Route
	BundleOutput   *BundleOutput
	GlobalNotFound string         // non-empty when a GlobalNotFound export exists
	CSSURLs        []string       // external stylesheet URLs
	DocumentProps  *DocumentProps // document-level props from root layout, or nil
}

// App is the fully-wired Pola application. It implements http.Handler.
type App struct {
	cfg       Config
	registry  *Registry
	handler   http.Handler
	artifacts BuildArtifacts
}

// New creates and builds a Pola application from the given options.
// All components are resolved from the DI container.
func New(opts ...Option) (*App, error) {
	// Populated by internal/pipeline — separate package to avoid cycles.
	// This stub exists so core can be imported without internal/.
	panic("core.New: not implemented — import github.com/polagonow/pola to get the full implementation")
}

// NewApp constructs an App with all fields populated. Called exclusively by
// internal/pipeline to avoid import cycles — end users should call pola.New.
func NewApp(cfg *Config, registry *Registry, handler http.Handler) *App {
	a := &App{
		registry: registry,
		handler:  handler,
	}
	if cfg != nil {
		a.cfg = *cfg
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

// Cache resolves the cache from the DI container on demand.
func (a *App) Cache() Cache {
	cache, err := Invoke[Cache](a.registry)
	if err != nil {
		return nil
	}
	return cache
}

// Build runs the build pipeline explicitly (called by New internally).
func (a *App) Build(ctx context.Context) error { return nil }

// Registry returns the service registry.
func (a *App) Registry() *Registry { return a.registry }

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

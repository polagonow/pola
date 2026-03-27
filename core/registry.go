package core

import (
	"context"
	"net/http"

	samberdo "github.com/samber/do/v2"
)

// Config holds resolved configuration for a Pola application.
// Constructed internally from Option functions — not created directly by users.
type Config struct {
	WebAppPath string
	PublicDir  string
	Dev        bool
}

// Option configures a Pola application.
type Option func(*Config)

// WithWebAppPath sets the path to the frontend app directory.
func WithWebAppPath(path string) Option {
	return func(c *Config) { c.WebAppPath = path }
}

// WithPublicDir sets where compiled assets are written.
func WithPublicDir(dir string) Option {
	return func(c *Config) { c.PublicDir = dir }
}

// WithDev enables development mode with hot reload.
func WithDev(dev bool) Option {
	return func(c *Config) { c.Dev = dev }
}

// BuildArtifacts holds the outputs produced during the build pipeline.
type BuildArtifacts struct {
	Output *BundleOutput
}

// PrebuildArtifacts holds all data pre-computed during mage Bundle so that the
// production binary (embed tag) can skip route scanning and JS bundling entirely.
type PrebuildArtifacts struct {
	Routes         []Route
	BundleOutput   *BundleOutput
	GlobalNotFound string   // non-empty when a GlobalNotFound export exists
	CSSURLs        []string // external stylesheet URLs
}

// App is the fully-wired Pola application. It implements http.Handler.
type App struct {
	cfg       Config
	injector  samberdo.Injector
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
func NewApp(cfg *Config, injector samberdo.Injector, handler http.Handler) *App {
	a := &App{
		injector: injector,
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
	cache, err := samberdo.Invoke[Cache](a.injector)
	if err != nil {
		return nil
	}
	return cache
}

// Build runs the build pipeline explicitly (called by New internally).
func (a *App) Build(ctx context.Context) error { return nil }

// Injector returns the DI container for advanced usage.
func (a *App) Injector() samberdo.Injector { return a.injector }

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

// Package pola is the main entry point for the Pola web framework.
//
// Usage:
//
//	func main() {
//	    if err := pola.Ready(); err != nil {
//	        log.Fatal(err)
//	    }
//	    log.Fatal(http.ListenAndServe(pola.Addr(), nil))
//	}
package pola

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/env"
	"github.com/polagonow/pola/internal"
)

// defaultApp holds the lazily-built default application instance.
var defaultApp struct {
	once    sync.Once
	app     *core.App
	env     *env.Env
	err     error
	plugins []core.Plugin
	opts    []core.Option
}

// Use registers plugins for the default app. Call before Ready().
// The generated pola_plugins.go calls this automatically.
func Use(plugins ...core.Plugin) {
	defaultApp.plugins = append(defaultApp.plugins, plugins...)
}

func serve(w http.ResponseWriter, r *http.Request) {
	defaultApp.once.Do(buildDefault)
	if defaultApp.err != nil {
		http.Error(w, "pola: "+defaultApp.err.Error(), http.StatusInternalServerError)
		return
	}
	defaultApp.app.ServeHTTP(w, r)
}

func buildDefault() {
	e, err := env.Load()
	if err != nil {
		defaultApp.err = err
		return
	}
	defaultApp.env = e
	opts := []core.Option{
		core.WithWebAppDir(e.WebAppPath),
		core.WithPublicDir(e.PublicDir),
		core.WithDev(e.IsDev()),
	}
	opts = append(opts, defaultApp.opts...)
	// Resolve config to determine the HTTP path.
	cfg := &core.Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	path := "/"
	if cfg.HTTPPath != "" {
		path = cfg.HTTPPath
	}
	http.Handle(path, http.HandlerFunc(serve))

	builder := core.NewAppBuilder(opts...)
	builder.Use(defaultApp.plugins...)
	defaultApp.app, defaultApp.err = internal.Build(context.Background(), builder)
}

// Ready eagerly builds the default app. Call from main() before
// ListenAndServe to avoid first-request latency. Safe to call multiple times.
//
// Optional core.Option overrides are applied after environment-derived
// defaults, so they take precedence.
//
// When POLA_BUILD_ONLY=true (set by `pola build` stage 1), Ready builds
// the app (which triggers asset bundling) and then exits the process.
func Ready(opts ...core.Option) error {
	defaultApp.opts = opts
	defaultApp.once.Do(buildDefault)
	if defaultApp.err != nil {
		return defaultApp.err
	}
	if defaultApp.env != nil && defaultApp.env.BuildOnly {
		fmt.Println("Build-only mode: assets bundled, exiting.")
		os.Exit(0)
	}
	return nil
}

// Addr returns the listen address from POLA_ADDRESS and PORT env vars
// (e.g. ":3000" or "localhost:3000"). Safe to call multiple times.
func Addr() string {
	defaultApp.once.Do(buildDefault)
	if defaultApp.env == nil {
		return ":3000"
	}
	return defaultApp.env.Address + ":" + defaultApp.env.Port
}

// Registry returns the DI registry of the default app.
// Must be called after Ready().
func Registry() *core.Registry {
	if defaultApp.app == nil {
		return nil
	}
	return defaultApp.app.Registry()
}

// NewApp creates an AppBuilder for explicit plugin-based construction.
//
//	builder := pola.NewApp(core.WithDev(true))
//	builder.Use(plugins...)
//	app, err := pola.BuildApp(ctx, builder)
func NewApp(opts ...core.Option) *core.AppBuilder {
	return core.NewAppBuilder(opts...)
}

// BuildApp builds an App from an AppBuilder.
func BuildApp(ctx context.Context, builder *core.AppBuilder) (*core.App, error) {
	return internal.Build(ctx, builder)
}

// ListenAndServe starts the default app with graceful shutdown support.
// On SIGINT/SIGTERM, in-flight requests are drained within the configured
// timeout (POLA_SHUTDOWN_TIMEOUT env var, default 30s) before the server
// stops. Returns nil on clean shutdown.
func ListenAndServe() error {
	addr := Addr()
	srv := &http.Server{Addr: addr}

	shutdownTimeout := 30 * time.Second
	if v := os.Getenv("POLA_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			shutdownTimeout = d
		}
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

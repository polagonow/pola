// Package pola is the main entry point for the Pola web framework.
//
// Import this package to automatically register a handler on
// http.DefaultServeMux — the same pattern as net/http/pprof:
//
//	import "github.com/polagonow/pola"
//
//	func main() {
//	    if err := pola.Ready(core.WithWebAppPath("app")); err != nil {
//	        log.Fatal(err)
//	    }
//	    log.Fatal(http.ListenAndServe(pola.Addr(), nil))
//	}
package pola

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/env"
	"github.com/polagonow/pola/internal"
)

// defaultApp holds the lazily-built default application instance.
var defaultApp struct {
	once sync.Once
	app  *core.App
	env  *env.Env
	err  error
	opts []core.Option
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
		core.WithWebAppPath(e.WebAppPath),
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
	defaultApp.app, defaultApp.err = New(opts...)
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

// New creates and builds a Pola application from the given options.
// Use this for explicit control over configuration; otherwise, use
// Ready() + http.DefaultServeMux for the zero-config path.
//
//	app, err := pola.New(core.WithWebAppPath("./app"), core.WithDev(true))
//	http.ListenAndServe(":8080", app)
func New(opts ...core.Option) (*core.App, error) {
	cfg := &core.Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return internal.Build(cfg)
}

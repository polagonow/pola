// Package pola is the main entry point for the Pola web framework.
// Import this package to get a fully wired application.
package pola

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/internal"
)

// New creates and builds a Pola application from the given options.
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

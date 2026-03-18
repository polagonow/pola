// Package htmx provides an HTMX-aware router stub for the Pola framework.
// It implements core.Router and is intended for applications that use HTMX
// for partial-page updates rather than full client-side React rendering.
package htmx

import (
	"context"

	"github.com/polagonow/pola/core"
)

// Router is an HTMX router stub.
// TODO: Implement HTMX-aware route scanning and partial-page update logic.
type Router struct{}

// New returns a new htmx Router.
func New() *Router { return &Router{} }

// Name returns the router name.
func (r *Router) Name() string { return "htmx" }

// ScanRoutes is a stub; returns nil routes.
func (r *Router) ScanRoutes(_ context.Context, _ core.FS, _ string, _ []string) ([]core.Route, error) {
	return nil, nil
}

// Resolve is a stub; always returns no match.
func (r *Router) Resolve(_ context.Context, _ string) (*core.Route, map[string]any) {
	return nil, nil
}

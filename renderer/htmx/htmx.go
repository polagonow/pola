// Package htmx provides an HTMX renderer stub for Pola.
package htmx

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by Render until the full HTMX implementation
// is wired in.
var ErrNotImplemented = errors.New("htmx renderer: not yet implemented")

// Renderer is the Pola renderer for HTMX-driven pages.
type Renderer struct{}

// New returns a Renderer stub.
func New() *Renderer { return &Renderer{} }

// Name implements core.Renderer.
func (r *Renderer) Name() string { return "htmx" }

// FileExtensions implements core.Renderer.
func (r *Renderer) FileExtensions() []string { return []string{".html", ".templ"} }

// Capabilities implements core.Renderer.
func (r *Renderer) Capabilities() []core.Capability { return []core.Capability{"server-side"} }

// Render implements core.Renderer.
func (r *Renderer) Render(_ context.Context, _ core.RenderRequest) (core.RenderResult, error) {
	return core.RenderResult{}, ErrNotImplemented
}

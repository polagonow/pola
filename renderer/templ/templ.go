// Package templ provides a Go Templ renderer stub for Pola.
// Full implementation requires github.com/a-h/templ.
package templ

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by Render until the full templ implementation
// is wired in.
var ErrNotImplemented = errors.New("templ renderer: not yet implemented")

// Renderer is the Pola renderer for Go Templ templates.
type Renderer struct{}

// New returns a Renderer stub.
func New() *Renderer { return &Renderer{} }

// Name implements core.Renderer.
func (r *Renderer) Name() string { return "templ" }

// FileExtensions implements core.Renderer.
func (r *Renderer) FileExtensions() []string { return []string{".templ"} }

// Capabilities implements core.Renderer.
func (r *Renderer) Capabilities() []core.Capability { return []core.Capability{"server-side"} }

// Render implements core.Renderer.
func (r *Renderer) Render(_ context.Context, _ core.RenderRequest) (core.RenderResult, error) {
	return core.RenderResult{}, ErrNotImplemented
}

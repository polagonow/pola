// Package angular provides an Angular renderer stub for Pola.
package angular

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by Render until the full Angular implementation
// is wired in.
var ErrNotImplemented = errors.New("angular renderer: not yet implemented")

// Renderer is the Pola renderer for Angular components.
type Renderer struct{}

// New returns a Renderer stub.
func New() *Renderer { return &Renderer{} }

// Name implements core.Renderer.
func (r *Renderer) Name() string { return "angular" }

// FileExtensions implements core.Renderer.
func (r *Renderer) FileExtensions() []string { return []string{".html", ".ts"} }

// Capabilities implements core.Renderer.
func (r *Renderer) Capabilities() []core.Capability { return []core.Capability{"server-side"} }

// Render implements core.Renderer.
func (r *Renderer) Render(_ context.Context, _ core.RenderRequest) (core.RenderResult, error) {
	return core.RenderResult{}, ErrNotImplemented
}

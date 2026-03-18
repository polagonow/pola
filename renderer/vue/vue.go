// Package vue provides a Vue.js renderer stub for Pola.
package vue

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by Render until the full Vue implementation
// is wired in.
var ErrNotImplemented = errors.New("vue renderer: not yet implemented")

// Renderer is the Pola renderer for Vue Single-File Components.
type Renderer struct{}

// New returns a Renderer stub.
func New() *Renderer { return &Renderer{} }

// Name implements core.Renderer.
func (r *Renderer) Name() string { return "vue" }

// FileExtensions implements core.Renderer.
func (r *Renderer) FileExtensions() []string { return []string{".vue"} }

// Capabilities implements core.Renderer.
func (r *Renderer) Capabilities() []core.Capability { return []core.Capability{"server-side"} }

// Render implements core.Renderer.
func (r *Renderer) Render(_ context.Context, _ core.RenderRequest) (core.RenderResult, error) {
	return core.RenderResult{}, ErrNotImplemented
}

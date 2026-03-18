// Package svelte provides a Svelte renderer stub for Pola.
package svelte

import (
	"context"
	"errors"

	"github.com/polagonow/pola/core"
)

// ErrNotImplemented is returned by Render until the full Svelte implementation
// is wired in.
var ErrNotImplemented = errors.New("svelte renderer: not yet implemented")

// Renderer is the Pola renderer for Svelte components.
type Renderer struct{}

// New returns a Renderer stub.
func New() *Renderer { return &Renderer{} }

// Name implements core.Renderer.
func (r *Renderer) Name() string { return "svelte" }

// FileExtensions implements core.Renderer.
func (r *Renderer) FileExtensions() []string { return []string{".svelte"} }

// Capabilities implements core.Renderer.
func (r *Renderer) Capabilities() []core.Capability { return []core.Capability{"server-side"} }

// Render implements core.Renderer.
func (r *Renderer) Render(_ context.Context, _ core.RenderRequest) (core.RenderResult, error) {
	return core.RenderResult{}, ErrNotImplemented
}

// Package templ provides a Go Templ renderer stub for Pola.
// Full implementation requires github.com/a-h/templ.
package templ

import (
	"fmt"
	"net/http"

	"github.com/polagonow/pola/core"
)

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

// ServeHTTP implements core.Renderer.
func (r *Renderer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, _, _, status, _ := core.RenderRequestFrom(req.Context())
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, "<html><body><h1>templ renderer: not yet implemented</h1></body></html>")
}

// Package pprof provides a toggleable pprof profiling endpoint.
// Import this package (blank import) to register /debug/pprof/ routes.
package pprof

import (
	"net/http"
	_ "net/http/pprof" // register pprof handlers on http.DefaultServeMux

	"github.com/polagonow/pola/core"
)

// server is the pprof endpoint server.
type server struct{}

// New creates a pprof server that mounts all standard /debug/pprof/ handlers.
func New() core.Pprof { return &server{} }

// Ensure server satisfies core.Pprof.
var _ core.Pprof = (*server)(nil)

// Name returns the pprof implementation name.
func (s *server) Name() string { return "pprof" }

// Path returns the pprof endpoint path prefix.
func (s *server) Path() string { return "/debug/pprof" }

// Handler returns an http.Handler that serves pprof endpoints under /debug/pprof/.
func (s *server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/", http.DefaultServeMux)
	return mux
}

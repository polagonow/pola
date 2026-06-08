// Package pprof provides a toggleable pprof profiling endpoint.
// Import this package (blank import) to register the profiling routes; they are
// served under the reserved /_pola/pprof prefix by default (override via
// POLA_PPROF_PATH).
package pprof

import (
	"cmp"
	"net/http"
	_ "net/http/pprof" // register pprof handlers on http.DefaultServeMux
	"os"
	"strings"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/reserved"
)

// server is the pprof endpoint server.
type server struct{ path string }

// New creates a pprof server. The endpoint prefix defaults to reserved.Pprof
// and may be overridden via POLA_PPROF_PATH.
func New() core.Pprof {
	return &server{path: cmp.Or(os.Getenv("POLA_PPROF_PATH"), reserved.Pprof)}
}

// Ensure server satisfies core.Pprof.
var _ core.Pprof = (*server)(nil)

// Name returns the pprof implementation name.
func (s *server) Name() string { return "pprof" }

// Path returns the pprof endpoint path prefix.
func (s *server) Path() string { return s.path }

// Handler returns an http.Handler that serves the standard pprof endpoints.
// net/http/pprof registers its handlers on http.DefaultServeMux under
// /debug/pprof/, so requests to the configured prefix are rewritten back to
// that namespace before being dispatched.
func (s *server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest, ok := strings.CutPrefix(r.URL.Path, s.path)
		if !ok {
			http.DefaultServeMux.ServeHTTP(w, r)
			return
		}
		if rest == "" {
			rest = "/"
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/debug/pprof" + rest
		http.DefaultServeMux.ServeHTTP(w, r2)
	})
}

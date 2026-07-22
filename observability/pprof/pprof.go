// Package pprof provides a toggleable pprof profiling endpoint.
// Import this package (blank import) to register the profiling routes; they are
// served under the reserved /_pola/pprof prefix by default (override via
// POLA_PPROF_PATH).
//
// SECURITY: pprof endpoints expose heap/goroutine/CPU profiles and command-line
// arguments, leaking sensitive internals about a running process. Never expose
// them publicly without an auth guard. Use [WithToken] or [WithBasicAuth] (or
// the POLA_PPROF_TOKEN / POLA_PPROF_USER + POLA_PPROF_PASS environment
// variables) to require a credential; when set, the handler returns 401 to
// unauthenticated requests.
package pprof

import (
	"cmp"
	"crypto/subtle"
	"net/http"
	_ "net/http/pprof" // register pprof handlers on http.DefaultServeMux
	"os"
	"strings"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/reserved"
)

// server is the pprof endpoint server.
type server struct {
	path string
	// auth, when non-nil, guards the pprof handler and returns 401 for requests
	// that fail the check. Nil means the endpoints are served unauthenticated
	// (the default, opt-in-to-secure behavior).
	auth func(*http.Request) bool
}

// Option configures a pprof server.
type Option func(*server)

// WithToken guards the pprof endpoints behind a bearer token. Requests must
// carry an "Authorization: Bearer <token>" header (or a "?token=<token>" query
// param) matching token; otherwise they receive 401. An empty token is a no-op.
func WithToken(token string) Option {
	return func(s *server) {
		if token == "" {
			return
		}
		s.auth = func(r *http.Request) bool {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got == "" {
				got = r.URL.Query().Get("token")
			}
			return secureEqual(got, token)
		}
	}
}

// WithBasicAuth guards the pprof endpoints behind HTTP Basic auth. Requests
// must supply matching credentials; otherwise they receive 401. Empty
// credentials are a no-op.
func WithBasicAuth(user, pass string) Option {
	return func(s *server) {
		if user == "" && pass == "" {
			return
		}
		s.auth = func(r *http.Request) bool {
			u, p, ok := r.BasicAuth()
			if !ok {
				return false
			}
			return secureEqual(u, user) && secureEqual(p, pass)
		}
	}
}

// secureEqual compares two strings in constant time to avoid leaking length or
// content via timing.
func secureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// New creates a pprof server. The endpoint prefix defaults to reserved.Pprof
// and may be overridden via POLA_PPROF_PATH.
//
// If no auth Option is supplied, the POLA_PPROF_TOKEN environment variable (or
// POLA_PPROF_USER + POLA_PPROF_PASS for basic auth) is honored so deployments
// can require a credential without code changes. When neither is configured the
// endpoints are served unauthenticated — see the package doc security note.
func New(opts ...Option) core.Pprof {
	s := &server{path: cmp.Or(os.Getenv("POLA_PPROF_PATH"), reserved.Pprof)}
	for _, opt := range opts {
		opt(s)
	}
	if s.auth == nil {
		if tok := os.Getenv("POLA_PPROF_TOKEN"); tok != "" {
			WithToken(tok)(s)
		} else if u, p := os.Getenv("POLA_PPROF_USER"), os.Getenv("POLA_PPROF_PASS"); u != "" || p != "" {
			WithBasicAuth(u, p)(s)
		}
	}
	return s
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
//
// When an auth guard is configured (see [New], [WithToken], [WithBasicAuth]),
// requests that fail the check are rejected with 401 before reaching pprof.
func (s *server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth != nil && !s.auth(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
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

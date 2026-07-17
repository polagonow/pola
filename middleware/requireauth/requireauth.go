// Package requireauth provides declarative route protection: requests to
// protected path prefixes from unauthenticated visitors are redirected away
// (typically to a sign-in page). It verifies the JWT session cookie (the
// pola-native session written by middleware/jwt) directly, so it does not depend
// on middleware ordering.
package requireauth

import (
	"net/http"
	"strings"

	authjwt "github.com/polagonow/pola/auth/jwt"
	"github.com/polagonow/pola/core"
)

// Option configures the route-protection middleware.
type Option func(*config)

type config struct {
	secret     []byte
	cookieName string
	paths      []string
	redirect   string
}

// WithSecret sets the HS256 secret used to verify the session cookie (usually
// derived from AUTH_SECRET — the same secret middleware/jwt signs with).
func WithSecret(secret []byte) Option { return func(c *config) { c.secret = secret } }

// WithCookieName sets the session cookie name (default "session").
func WithCookieName(name string) Option { return func(c *config) { c.cookieName = name } }

// WithProtect adds protected path prefixes. A pattern matches the exact path or
// any path beneath it; a trailing "/*" is accepted and ignored.
func WithProtect(paths ...string) Option {
	return func(c *config) { c.paths = append(c.paths, paths...) }
}

// WithRedirect sets where unauthenticated visitors are redirected (default "/").
func WithRedirect(url string) Option { return func(c *config) { c.redirect = url } }

type mw struct{ cfg config }

// New creates the route-protection middleware.
func New(opts ...Option) core.Middleware {
	cfg := config{cookieName: "session", redirect: "/"}
	for _, o := range opts {
		o(&cfg)
	}
	return &mw{cfg: cfg}
}

func (m *mw) Name() string { return "require-auth" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.isProtected(r.URL.Path) && !m.authenticated(r) {
			http.Redirect(w, r, m.cfg.redirect, http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *mw) isProtected(path string) bool {
	for _, p := range m.cfg.paths {
		if matchPrefix(path, p) {
			return true
		}
	}
	return false
}

// matchPrefix reports whether path is pattern or lives beneath it. "/dashboard"
// matches "/dashboard" and "/dashboard/x" but NOT "/dashboardxyz".
func matchPrefix(path, pattern string) bool {
	pattern = strings.TrimSuffix(pattern, "*")
	pattern = strings.TrimSuffix(pattern, "/")
	if pattern == "" {
		return false
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func (m *mw) authenticated(r *http.Request) bool {
	if len(m.cfg.secret) == 0 {
		return false // fail closed: without a secret we cannot verify the session
	}
	c, err := r.Cookie(m.cfg.cookieName)
	if err != nil || c.Value == "" {
		return false
	}
	_, err = authjwt.Verify(c.Value, m.cfg.secret)
	return err == nil
}

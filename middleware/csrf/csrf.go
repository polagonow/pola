// Package csrf provides CSRF protection middleware using gorilla/csrf.
package csrf

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"

	gorillacsrf "github.com/gorilla/csrf"

	"github.com/polagonow/pola/core"
)

// Option configures the CSRF middleware.
type Option func(*config)

type config struct {
	authKey    []byte
	secure     bool
	cookieName string
	exempt     []string
}

// WithAuthKey sets the 32-byte authentication key used for CSRF tokens.
// If not provided, a random key is generated at startup.
func WithAuthKey(key []byte) Option {
	return func(c *config) { c.authKey = key }
}

// WithSecure controls the Secure flag on the CSRF cookie.
// Defaults to true (production). Set to false for HTTP-only development.
func WithSecure(secure bool) Option {
	return func(c *config) { c.secure = secure }
}

// WithCookieName sets the name of the CSRF cookie.
func WithCookieName(name string) Option {
	return func(c *config) { c.cookieName = name }
}

// WithExempt registers request paths that bypass CSRF protection entirely.
// This is required for endpoints that cannot present a CSRF token, such as
// third-party webhooks (e.g. Stripe's "/api/stripe/webhook"), which authenticate
// requests via their own signature instead.
//
// A pattern matches when the request path equals it exactly, or — when the
// pattern ends with "/*" — when the request path is under that prefix. For
// example "/api/webhooks/*" exempts "/api/webhooks/stripe" and everything below
// it. Patterns accumulate across calls.
func WithExempt(paths ...string) Option {
	return func(c *config) { c.exempt = append(c.exempt, paths...) }
}

type mw struct {
	protect   func(http.Handler) http.Handler
	plaintext bool // true when Secure=false (dev over HTTP)
	exempt    []string
}

// isExempt reports whether path matches any configured exemption pattern.
func (m *mw) isExempt(path string) bool {
	for _, p := range m.exempt {
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/*") {
			prefix := strings.TrimSuffix(p, "*") // keep trailing slash
			if strings.HasPrefix(path, prefix) {
				return true
			}
			// Also treat "/api/foo/*" as matching the bare "/api/foo".
			if path == strings.TrimSuffix(prefix, "/") {
				return true
			}
			continue
		}
		if path == p {
			return true
		}
	}
	return false
}

// New creates a CSRF protection middleware.
func New(opts ...Option) core.Middleware {
	cfg := &config{
		secure:     true,
		cookieName: "_csrf",
	}
	for _, o := range opts {
		o(cfg)
	}

	if len(cfg.authKey) == 0 {
		cfg.authKey = make([]byte, 32)
		if _, err := rand.Read(cfg.authKey); err != nil {
			panic("csrf: failed to generate auth key: " + err.Error())
		}
	}

	protect := gorillacsrf.Protect(
		cfg.authKey,
		gorillacsrf.Secure(cfg.secure),
		gorillacsrf.CookieName(cfg.cookieName),
		gorillacsrf.SameSite(gorillacsrf.SameSiteLaxMode),
	)

	return &mw{protect: protect, plaintext: !cfg.secure, exempt: cfg.exempt}
}

func (m *mw) Name() string { return "csrf" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	// Wrap next to inject the CSRF token into a response header so
	// client-side JS can read it for subsequent requests. The token is
	// read inside the inner handler because gorilla/csrf populates it
	// in the request context within Protect, before calling next.
	tokenInjector := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := gorillacsrf.Token(r)
		w.Header().Set("X-CSRF-Token", token)
		// Store the token in the request context so the orchestrator can
		// inject it into the HTML shell as <meta> tags.
		ctx := context.WithValue(r.Context(), core.CSRFTokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
	var protected http.Handler = m.protect(tokenInjector)
	if m.plaintext {
		// In dev mode (HTTP), tell gorilla/csrf to use http:// scheme for
		// origin checking instead of defaulting to https://.
		inner := protected
		protected = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inner.ServeHTTP(w, gorillacsrf.PlaintextHTTPRequest(r))
		})
	}
	if len(m.exempt) == 0 {
		return protected
	}
	// Exempt paths bypass gorilla/csrf entirely (no token required). They still
	// pass through next so downstream handlers run normally.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.isExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		protected.ServeHTTP(w, r)
	})
}

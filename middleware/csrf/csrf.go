// Package csrf provides CSRF protection middleware using gorilla/csrf.
package csrf

import (
	"context"
	"crypto/rand"
	"net/http"

	gorillacsrf "github.com/gorilla/csrf"

	"github.com/polagonow/pola/core"
)

// Option configures the CSRF middleware.
type Option func(*config)

type config struct {
	authKey    []byte
	secure     bool
	cookieName string
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

type mw struct {
	protect   func(http.Handler) http.Handler
	plaintext bool // true when Secure=false (dev over HTTP)
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

	return &mw{protect: protect, plaintext: !cfg.secure}
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
	protected := m.protect(tokenInjector)
	if m.plaintext {
		// In dev mode (HTTP), tell gorilla/csrf to use http:// scheme for
		// origin checking instead of defaulting to https://.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			protected.ServeHTTP(w, gorillacsrf.PlaintextHTTPRequest(r))
		})
	}
	return protected
}

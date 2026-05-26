// Package securityheaders provides a Middleware that sets common HTTP security headers.
package securityheaders

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/polagonow/pola/core"
)

// Option configures the security headers middleware.
type Option func(*mw)

// WithDev relaxes Content-Security-Policy to allow inline scripts,
// which is required for hot-reload in development mode.
func WithDev() Option {
	return func(m *mw) { m.dev = true }
}

type mw struct{ dev bool }

// New creates a security headers middleware.
func New(opts ...Option) core.Middleware {
	m := &mw{}
	for _, o := range opts {
		o(m)
	}
	return m
}

func (m *mw) Name() string { return "securityheaders" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a per-request nonce for inline scripts.
		nonce := generateNonce()
		ctx := context.WithValue(r.Context(), core.NonceContextKey, nonce)
		r = r.WithContext(ctx)

		csp := fmt.Sprintf("default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' 'unsafe-inline'", nonce)
		if m.dev {
			csp = fmt.Sprintf("default-src 'self'; script-src 'self' 'unsafe-inline' 'nonce-%s'; style-src 'self' 'unsafe-inline'", nonce)
		}

		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", csp)
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// generateNonce returns a cryptographically random base64-encoded nonce.
func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("securityheaders: failed to generate nonce: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

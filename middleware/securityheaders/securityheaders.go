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

// defaultHSTS is a conservative production HSTS policy: one year, subdomains
// included. It is intentionally omitted in dev mode and on plain-HTTP requests.
const defaultHSTS = "max-age=31536000; includeSubDomains"

// Option configures the security headers middleware.
type Option func(*mw)

// WithDev relaxes Content-Security-Policy to allow inline scripts,
// which is required for hot-reload in development mode. It also disables HSTS,
// since dev typically runs over plain HTTP.
func WithDev() Option {
	return func(m *mw) { m.dev = true }
}

// WithHSTS sets the Strict-Transport-Security header value (e.g.
// "max-age=31536000; includeSubDomains; preload"). Pass an empty string to
// disable HSTS entirely. HSTS is only emitted on requests served over HTTPS and
// never in dev mode, so it will not break local plain-HTTP development.
func WithHSTS(value string) Option {
	return func(m *mw) { m.hsts = value }
}

type mw struct {
	dev  bool
	hsts string
}

// New creates a security headers middleware.
func New(opts ...Option) core.Middleware {
	m := &mw{hsts: defaultHSTS}
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

		// style-src keeps 'unsafe-inline': many templates/components emit inline
		// style="" attributes, and nonce-ing every one would break rendering.
		// This is a known, minor CSP weakness (it permits inline CSS, not JS);
		// tighten it via a future WithStyleSrc option if your app allows it.
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
		// HSTS only makes sense over HTTPS; setting it on plain-HTTP dev could
		// wedge a browser onto https://localhost. Emit it only for secure
		// requests and never in dev mode.
		if m.hsts != "" && !m.dev && isHTTPS(r) {
			h.Set("Strict-Transport-Security", m.hsts)
		}
		next.ServeHTTP(w, r)
	})
}

// isHTTPS reports whether the request was served over TLS, honoring the
// X-Forwarded-Proto header set by TLS-terminating proxies/load balancers.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

// generateNonce returns a cryptographically random base64-encoded nonce.
func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("securityheaders: failed to generate nonce: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

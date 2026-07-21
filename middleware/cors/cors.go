// Package cors provides a Middleware implementing Cross-Origin Resource Sharing
// (CORS), including preflight (OPTIONS) handling. It fills a gap in pola's
// middleware set — there was no built-in CORS — which apps with a separately
// hosted frontend or public API need.
//
// Defaults are permissive-but-safe for development (any origin, common methods,
// reflected request headers) and are tightened for production by supplying
// options:
//
//	pola.Use(cors.Plugin(
//	    cors.WithAllowedOrigins("https://app.example.com"),
//	    cors.WithAllowCredentials(),
//	))
package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/polagonow/pola/core"
)

// Config is the resolved CORS policy. Construct it via [New]/[Plugin] and the
// With* options rather than by hand.
type Config struct {
	// AllowedOrigins lists permitted origins. "*" allows any. Matched
	// case-insensitively against the request's Origin header.
	AllowedOrigins []string
	// AllowedMethods is advertised in preflight responses.
	AllowedMethods []string
	// AllowedHeaders is advertised in preflight responses. "*" reflects the
	// browser's Access-Control-Request-Headers (required when credentials are
	// allowed, since a literal "*" is invalid in that case).
	AllowedHeaders []string
	// ExposedHeaders is advertised to the browser as readable on responses.
	ExposedHeaders []string
	// AllowCredentials allows cookies/authorization on cross-origin requests.
	// When set, a wildcard origin is echoed back as the concrete origin.
	AllowCredentials bool
	// MaxAge caps how long (seconds) a preflight result may be cached. 0 omits
	// the header.
	MaxAge int
}

// Default returns the development-friendly policy used when no options are
// given: any origin, the common methods, and reflected request headers.
func Default() Config {
	return Config{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowedHeaders: []string{"*"},
	}
}

// Option configures the CORS policy.
type Option func(*Config)

// WithAllowedOrigins sets the permitted origins (replacing the default "*").
func WithAllowedOrigins(origins ...string) Option {
	return func(c *Config) { c.AllowedOrigins = origins }
}

// WithAllowedMethods sets the advertised methods.
func WithAllowedMethods(methods ...string) Option {
	return func(c *Config) { c.AllowedMethods = methods }
}

// WithAllowedHeaders sets the advertised request headers.
func WithAllowedHeaders(headers ...string) Option {
	return func(c *Config) { c.AllowedHeaders = headers }
}

// WithExposedHeaders sets the headers the browser may read on responses.
func WithExposedHeaders(headers ...string) Option {
	return func(c *Config) { c.ExposedHeaders = headers }
}

// WithAllowCredentials permits credentialed cross-origin requests.
func WithAllowCredentials() Option {
	return func(c *Config) { c.AllowCredentials = true }
}

// WithMaxAge sets the preflight cache duration in seconds.
func WithMaxAge(seconds int) Option {
	return func(c *Config) { c.MaxAge = seconds }
}

type mw struct {
	cfg             Config
	allowAllOrigins bool
	allowAllHeaders bool
	methods         string
	headers         string
	exposed         string
}

// New builds a CORS middleware from the given options.
func New(opts ...Option) core.Middleware {
	cfg := Default()
	for _, o := range opts {
		o(&cfg)
	}
	m := &mw{cfg: cfg}
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			m.allowAllOrigins = true
		}
	}
	for _, h := range cfg.AllowedHeaders {
		if h == "*" {
			m.allowAllHeaders = true
		}
	}
	m.methods = strings.Join(cfg.AllowedMethods, ", ")
	if !m.allowAllHeaders {
		m.headers = strings.Join(cfg.AllowedHeaders, ", ")
	}
	m.exposed = strings.Join(cfg.ExposedHeaders, ", ")
	return m
}

func (m *mw) Name() string { return "cors" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r) // not a cross-origin request
			return
		}

		h := w.Header()
		if !m.originAllowed(origin) {
			// Origin not permitted: emit no CORS headers (the browser blocks the
			// response). Still short-circuit preflight so it doesn't fall through
			// to a 404/handler.
			if isPreflight(r) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if m.allowAllOrigins && !m.cfg.AllowCredentials {
			h.Set("Access-Control-Allow-Origin", "*")
		} else {
			// A concrete origin must be echoed (never "*") when credentials are
			// allowed; Vary: Origin keeps caches per-origin.
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
		}
		if m.cfg.AllowCredentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}

		if isPreflight(r) {
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
			h.Set("Access-Control-Allow-Methods", m.methods)
			if m.allowAllHeaders {
				if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
					h.Set("Access-Control-Allow-Headers", reqHeaders)
				}
			} else if m.headers != "" {
				h.Set("Access-Control-Allow-Headers", m.headers)
			}
			if m.cfg.MaxAge > 0 {
				h.Set("Access-Control-Max-Age", strconv.Itoa(m.cfg.MaxAge))
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if m.exposed != "" {
			h.Set("Access-Control-Expose-Headers", m.exposed)
		}
		next.ServeHTTP(w, r)
	})
}

func (m *mw) originAllowed(origin string) bool {
	if m.allowAllOrigins {
		return true
	}
	for _, o := range m.cfg.AllowedOrigins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}

// isPreflight reports whether r is a CORS preflight request: an OPTIONS carrying
// the Access-Control-Request-Method header the browser adds.
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

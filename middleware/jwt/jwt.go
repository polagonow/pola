// Package jwt provides a stateless JWT-cookie session middleware — the
// pola-native equivalent of "email/password auth with a JWT stored to a cookie".
// It mirrors middleware/session's request-scoped API (Set/Get/Clear on the
// context) but stores a signed HS256 token in the cookie instead of a
// server-managed gorilla session.
package jwt

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"time"

	authjwt "github.com/polagonow/pola/auth/jwt"
	"github.com/polagonow/pola/core"
)

// Option configures the JWT middleware.
type Option func(*config)

type config struct {
	secret     []byte
	cookieName string
	expiry     time.Duration
	secure     bool
	httpOnly   bool
	path       string
	sameSite   http.SameSite
	refresh    bool
}

// WithSecret sets the HS256 signing secret (typically derived from AUTH_SECRET).
func WithSecret(secret []byte) Option { return func(c *config) { c.secret = secret } }

// WithCookieName sets the session cookie name (default "session").
func WithCookieName(name string) Option { return func(c *config) { c.cookieName = name } }

// WithExpiry sets the token lifetime (default 24h).
func WithExpiry(d time.Duration) Option { return func(c *config) { c.expiry = d } }

// WithExpiryString sets the token lifetime from a Go duration literal (e.g.
// "24h"). Invalid values are ignored, leaving the default.
func WithExpiryString(s string) Option {
	return func(c *config) {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			c.expiry = d
		}
	}
}

// WithSecure controls the Secure flag on the cookie. Default true; set false for
// HTTP-only development.
func WithSecure(secure bool) Option { return func(c *config) { c.secure = secure } }

// WithRefresh controls sliding expiration: when true (default), a valid session
// is re-issued with a fresh expiry on each response.
func WithRefresh(refresh bool) Option { return func(c *config) { c.refresh = refresh } }

type ctxKeyType struct{}

var ctxKey = ctxKeyType{}

// ctxData holds the per-request session state: the claims decoded from the
// incoming cookie plus any pending Set/Clear the handler performed.
type ctxData struct {
	claims  map[string]any // decoded from the incoming cookie (nil if none/invalid)
	pending map[string]any // claims to write via Set
	set     bool
	clear   bool
}

type mw struct {
	cfg config
}

// New creates the JWT session middleware.
func New(opts ...Option) core.Middleware {
	cfg := config{
		cookieName: "session",
		expiry:     24 * time.Hour,
		secure:     true,
		httpOnly:   true,
		path:       "/",
		sameSite:   http.SameSiteLaxMode,
		refresh:    true,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if len(cfg.secret) == 0 {
		// Without a configured secret, sign with an ephemeral key so dev still
		// works — but warn loudly: sessions won't survive a restart AND any code
		// that verifies the cookie with a different secret (e.g. route
		// protection reading AUTH_SECRET) will reject every session.
		fmt.Fprintln(os.Stderr,
			"pola/jwt: WARNING no signing secret set (AUTH_SECRET is empty). "+
				"Using a random ephemeral key — sessions will not persist across restarts "+
				"and route protection will reject all sessions. Set AUTH_SECRET.")
		cfg.secret = make([]byte, 32)
		_, _ = rand.Read(cfg.secret)
	}
	return &mw{cfg: cfg}
}

func (m *mw) Name() string { return "jwt" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cd := &ctxData{}
		if c, err := r.Cookie(m.cfg.cookieName); err == nil && c.Value != "" {
			if claims, err := authjwt.Verify(c.Value, m.cfg.secret); err == nil {
				cd.claims = claims
			}
		}
		sw := &saveWriter{ResponseWriter: w, cd: cd, cfg: &m.cfg}
		ctx := context.WithValue(r.Context(), ctxKey, cd)
		next.ServeHTTP(sw, r.WithContext(ctx))
		sw.save()
	})
}

// saveWriter writes the session cookie (Set-Cookie) before the response body is
// flushed, mirroring middleware/session's saveWriter.
type saveWriter struct {
	http.ResponseWriter
	cd    *ctxData
	cfg   *config
	saved bool
}

func (sw *saveWriter) save() {
	if sw.saved {
		return
	}
	sw.saved = true
	switch {
	case sw.cd.clear:
		sw.setCookie("", -1)
	case sw.cd.set:
		if tok, err := authjwt.Sign(sw.cd.pending, sw.cfg.secret, sw.cfg.expiry); err == nil {
			sw.setCookie(tok, int(sw.cfg.expiry.Seconds()))
		}
	case sw.cfg.refresh && sw.cd.claims != nil:
		// Sliding expiration: re-issue the existing session with a fresh exp.
		refreshed := make(map[string]any, len(sw.cd.claims))
		for k, v := range sw.cd.claims {
			if k == "iat" || k == "exp" {
				continue
			}
			refreshed[k] = v
		}
		if tok, err := authjwt.Sign(refreshed, sw.cfg.secret, sw.cfg.expiry); err == nil {
			sw.setCookie(tok, int(sw.cfg.expiry.Seconds()))
		}
	}
}

func (sw *saveWriter) setCookie(value string, maxAge int) {
	http.SetCookie(sw.ResponseWriter, &http.Cookie{
		Name:     sw.cfg.cookieName,
		Value:    value,
		Path:     sw.cfg.path,
		MaxAge:   maxAge,
		HttpOnly: sw.cfg.httpOnly,
		Secure:   sw.cfg.secure,
		SameSite: sw.cfg.sameSite,
	})
}

func (sw *saveWriter) WriteHeader(code int) {
	sw.save()
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *saveWriter) Write(b []byte) (int, error) {
	sw.save()
	return sw.ResponseWriter.Write(b)
}

func getCtx(ctx context.Context) *ctxData {
	cd, _ := ctx.Value(ctxKey).(*ctxData)
	return cd
}

// Get returns the current session claims — the ones just Set this request if
// any, otherwise the claims decoded from the incoming cookie. Returns an empty
// map when there is no session.
func Get(ctx context.Context) map[string]any {
	cd := getCtx(ctx)
	if cd == nil {
		return map[string]any{}
	}
	if cd.set {
		return cd.pending
	}
	if cd.claims == nil {
		return map[string]any{}
	}
	return cd.claims
}

// Claim is a convenience accessor for a single claim.
func Claim(ctx context.Context, key string) (any, bool) {
	v, ok := Get(ctx)[key]
	return v, ok
}

// Set establishes (or replaces) the session — the claims are signed into the
// cookie on the response. Pass e.g. map[string]any{"user": map[string]any{"id": id}}.
func Set(ctx context.Context, claims map[string]any) {
	cd := getCtx(ctx)
	if cd == nil {
		return
	}
	cd.pending = claims
	cd.set = true
	cd.clear = false
}

// Clear deletes the session cookie (sign-out).
func Clear(ctx context.Context) {
	cd := getCtx(ctx)
	if cd == nil {
		return
	}
	cd.clear = true
	cd.set = false
	cd.pending = nil
	cd.claims = nil
}

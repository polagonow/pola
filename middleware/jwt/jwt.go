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

// defaultMaxLifetime caps how long a session may be extended by sliding refresh,
// measured from the original "iat". Once exceeded, refresh stops re-issuing so a
// stolen or forgotten session can't live forever.
const defaultMaxLifetime = 7 * 24 * time.Hour

// Option configures the JWT middleware.
type Option func(*config)

type config struct {
	secret      []byte
	cookieName  string
	expiry      time.Duration
	secure      bool
	httpOnly    bool
	path        string
	sameSite    http.SameSite
	refresh     bool
	maxLifetime time.Duration
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
// is re-issued with a fresh expiry on each response — but never beyond the
// absolute max lifetime (see [WithMaxLifetime]).
func WithRefresh(refresh bool) Option { return func(c *config) { c.refresh = refresh } }

// WithMaxLifetime caps the total age a session may reach via sliding refresh,
// measured from the original "iat". Once a session is older than this, refresh
// stops extending it (the session expires at its current exp). Default 7 days.
// A non-positive value is ignored, leaving the default.
func WithMaxLifetime(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.maxLifetime = d
		}
	}
}

// iat0Claim carries the original (first) issued-at as a Unix timestamp. Unlike
// the standard "iat" — which Sign rewrites to now on every refresh — this claim
// is preserved across sliding refreshes so the absolute max lifetime is anchored
// to when the session was first established.
const iat0Claim = "iat0"

// originalIssuedAt returns the session's original issued-at (Unix seconds). It
// prefers iat0Claim; for sessions predating it, it falls back to the standard
// "iat" (both may be decoded as float64 by encoding/json).
func originalIssuedAt(claims map[string]any) int64 {
	if v, ok := asUnix(claims[iat0Claim]); ok {
		return v
	}
	if v, ok := asUnix(claims["iat"]); ok {
		return v
	}
	return time.Now().Unix()
}

func asUnix(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	}
	return 0, false
}

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
		cookieName:  "session",
		expiry:      24 * time.Hour,
		secure:      true,
		httpOnly:    true,
		path:        "/",
		sameSite:    http.SameSiteLaxMode,
		refresh:     true,
		maxLifetime: defaultMaxLifetime,
	}
	for _, o := range opts {
		o(&cfg)
	}
	switch {
	case len(cfg.secret) == 0:
		// Without a configured secret, sign with an ephemeral key so dev still
		// works — but warn loudly: sessions won't survive a restart AND any code
		// that verifies the cookie with a different secret (e.g. route
		// protection reading AUTH_SECRET) will reject every session.
		fmt.Fprintln(os.Stderr,
			"pola/jwt: WARNING no signing secret set (AUTH_SECRET is empty). "+
				"Using a random ephemeral key — sessions will not persist across restarts "+
				"and route protection will reject all sessions. Set AUTH_SECRET.")
		cfg.secret = make([]byte, authjwt.MinSecretLen)
		_, _ = rand.Read(cfg.secret)
	case len(cfg.secret) < authjwt.MinSecretLen:
		// A too-short secret is worse than none: it would silently sign and
		// verify weak tokens. Refuse to start rather than degrade security.
		panic(fmt.Sprintf("pola/jwt: signing secret must be at least %d bytes (got %d); "+
			"set AUTH_SECRET to a strong value", authjwt.MinSecretLen, len(cfg.secret)))
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
		// A freshly established session anchors its absolute lifetime at now:
		// stamp the original issued-at so later refreshes can enforce the cap.
		pending := make(map[string]any, len(sw.cd.pending)+1)
		for k, v := range sw.cd.pending {
			pending[k] = v
		}
		pending[iat0Claim] = time.Now().Unix()
		if tok, err := authjwt.Sign(pending, sw.cfg.secret, sw.cfg.expiry); err == nil {
			sw.setCookie(tok, int(sw.cfg.expiry.Seconds()))
		}
	case sw.cfg.refresh && sw.cd.claims != nil:
		// Sliding expiration: re-issue the existing session with a fresh exp,
		// preserving the original issued-at (iat0Claim) so the session can't be
		// extended indefinitely. Stop refreshing once the absolute max lifetime
		// is exceeded — the session then expires at its existing exp.
		iat0 := originalIssuedAt(sw.cd.claims)
		if sw.cfg.maxLifetime > 0 && time.Since(time.Unix(iat0, 0)) > sw.cfg.maxLifetime {
			return
		}
		refreshed := make(map[string]any, len(sw.cd.claims))
		for k, v := range sw.cd.claims {
			if k == "iat" || k == "exp" {
				continue
			}
			refreshed[k] = v
		}
		refreshed[iat0Claim] = iat0
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
	if _, ok := cd.claims[iat0Claim]; ok {
		// Hide the internal absolute-lifetime anchor from callers.
		out := make(map[string]any, len(cd.claims)-1)
		for k, v := range cd.claims {
			if k != iat0Claim {
				out[k] = v
			}
		}
		return out
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

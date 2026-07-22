// Package auth provides a pluggable authentication abstraction for API routes:
// an [Authenticator] extracts and validates credentials from a request and
// returns the authenticated user, a [Middleware] runs an authenticator and
// injects the user into the request context, and [UserFromContext] reads it back
// in handlers. Concrete strategies ([JWTAuthenticator], [BasicAuthenticator])
// build on the existing auth/jwt and auth/password primitives.
//
// It complements — not replaces — middleware/requireauth, which is oriented at
// redirecting unauthenticated *page* visitors to a sign-in screen. This package
// is for *API* auth: verify a credential, load the current user, expose it to
// the handler, and answer 401 when it's missing or invalid.
//
//	users := myUserService{}                     // implements auth.UserService[User]
//	authenticator := &auth.JWTAuthenticator[User]{Users: users, Secret: secret}
//	reg.AddMiddleware(auth.Middleware(authenticator))
//
//	func (c *Controller) Me(ctx core.Context) error {
//	    u, ok := auth.UserFromContext[User](ctx.Ctx())
//	    if !ok { return ctx.NoContent(http.StatusUnauthorized) }
//	    return ctx.JSON(http.StatusOK, u)
//	}
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	authjwt "github.com/polagonow/pola/auth/jwt"
	"github.com/polagonow/pola/core"
)

// Sentinel errors an [Authenticator] returns. Callers should treat both as an
// opaque 401 and avoid leaking which one occurred (so an attacker can't
// distinguish "no such user" from "wrong password").
var (
	// ErrNoCredentials means the request carried no credentials at all.
	ErrNoCredentials = errors.New("auth: no credentials")
	// ErrInvalidCredentials means credentials were present but rejected.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
)

// UserService loads the account behind a credential. T is the app's user type
// (a model or DTO). It is the single dependency the built-in authenticators
// need from the application.
type UserService[T any] interface {
	// FindByUsername returns the user identified by username (email, handle,
	// subject claim, …), or an error when none matches.
	FindByUsername(ctx context.Context, username string) (*T, error)
}

// Authenticator extracts and validates credentials from a request and returns
// the authenticated user, or an error explaining why authentication failed.
type Authenticator[T any] interface {
	Authenticate(r *http.Request) (*T, error)
}

// userCtxKey is the private context key under which the authenticated user is
// stored. A single key is used per app; UserFromContext recovers the concrete
// type via a type assertion.
type userCtxKey struct{}

// WithUser returns a copy of ctx carrying u as the authenticated user. Exposed
// so tests and custom middleware can inject a user directly.
func WithUser[T any](ctx context.Context, u *T) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

// UserFromContext returns the authenticated user injected by [Middleware], or
// nil and false when the request is unauthenticated (or authenticated as a
// different type).
func UserFromContext[T any](ctx context.Context) (*T, bool) {
	u, ok := ctx.Value(userCtxKey{}).(*T)
	return u, ok
}

type config struct {
	optional     bool
	unauthorized func(http.ResponseWriter, *http.Request, error)
}

// Option configures [Middleware].
type Option func(*config)

// WithOptional lets unauthenticated requests continue (with no user in context)
// instead of being rejected — useful for endpoints that behave differently when
// a user is present but don't require one.
func WithOptional() Option {
	return func(c *config) { c.optional = true }
}

// WithUnauthorized overrides the 401 response written when authentication fails.
func WithUnauthorized(fn func(http.ResponseWriter, *http.Request, error)) Option {
	return func(c *config) { c.unauthorized = fn }
}

type mw[T any] struct {
	auth Authenticator[T]
	config
}

// Middleware runs a through authenticator a on every request: on success the
// user is injected into the request context (read it with [UserFromContext]);
// on failure a 401 is written unless [WithOptional] was set.
func Middleware[T any](a Authenticator[T], opts ...Option) core.Middleware {
	c := config{unauthorized: defaultUnauthorized}
	for _, o := range opts {
		o(&c)
	}
	return &mw[T]{auth: a, config: c}
}

func (m *mw[T]) Name() string { return "auth" }

func (m *mw[T]) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := m.auth.Authenticate(r)
		if err != nil {
			if m.optional {
				next.ServeHTTP(w, r)
				return
			}
			m.unauthorized(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), u)))
	})
}

func defaultUnauthorized(w http.ResponseWriter, _ *http.Request, _ error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}

// IssueToken signs a JWT whose "sub" claim identifies the user, for use in a
// login handler after credentials have been verified. extra adds custom claims.
// The returned token validates against [JWTAuthenticator] using the same secret.
//
// It returns an error when secret is shorter than 32 bytes or expiry is not
// strictly positive (see auth/jwt.Sign) — pola never issues weakly-signed or
// non-expiring tokens.
func IssueToken(subject string, secret []byte, expiry time.Duration, extra map[string]any) (string, error) {
	claims := map[string]any{"sub": subject}
	for k, v := range extra {
		claims[k] = v
	}
	return authjwt.Sign(claims, secret, expiry)
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header,
// or "" when absent/malformed.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

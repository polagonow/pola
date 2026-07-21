package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/polagonow/pola/core"
)

// routeCtx is a minimal core.Context exercising the methods RouteMiddleware
// uses (Request/Writer/Ctx/SetContext); the embedded nil Context satisfies the
// rest, which are never called here.
type routeCtx struct {
	core.Context
	r *http.Request
	w *httptest.ResponseRecorder
}

func (c *routeCtx) Request() *http.Request         { return c.r }
func (c *routeCtx) Writer() http.ResponseWriter    { return c.w }
func (c *routeCtx) Ctx() context.Context           { return c.r.Context() }
func (c *routeCtx) SetContext(ctx context.Context) { c.r = c.r.WithContext(ctx) }

func newRouteCtx(token string) *routeCtx {
	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return &routeCtx{r: r, w: httptest.NewRecorder()}
}

func TestRouteMiddlewareInjectsUser(t *testing.T) {
	a := &JWTAuthenticator[user]{Users: newUsers(t), Secret: secret}
	tok, _ := IssueToken("ada", secret, time.Hour, nil)

	var seen string
	next := func(c core.Context) error {
		if u, ok := UserFromContext[user](c.Ctx()); ok {
			seen = u.Username
		}
		return nil
	}

	ctx := newRouteCtx(tok)
	if err := RouteMiddleware(a)(next)(ctx); err != nil {
		t.Fatalf("route middleware: %v", err)
	}
	if seen != "ada" {
		t.Errorf("handler saw user %q, want ada (injected via SetContext)", seen)
	}
}

func TestRouteMiddlewareRejects(t *testing.T) {
	a := &JWTAuthenticator[user]{Users: newUsers(t), Secret: secret}
	reached := false
	next := func(core.Context) error { reached = true; return nil }

	ctx := newRouteCtx("") // no token
	if err := RouteMiddleware(a)(next)(ctx); err != nil {
		t.Fatalf("route middleware returned err (should write 401 itself): %v", err)
	}
	if reached {
		t.Error("handler should not run on failed auth")
	}
	if ctx.w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", ctx.w.Code)
	}
}

func TestRouteMiddlewareOptional(t *testing.T) {
	a := &JWTAuthenticator[user]{Users: newUsers(t), Secret: secret}
	reached := false
	next := func(c core.Context) error {
		reached = true
		if _, ok := UserFromContext[user](c.Ctx()); ok {
			t.Error("no user expected on an optional unauthenticated request")
		}
		return nil
	}

	ctx := newRouteCtx("")
	_ = RouteMiddleware(a, WithOptional())(next)(ctx)
	if !reached {
		t.Error("optional route middleware should call next")
	}
}

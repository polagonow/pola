package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/polagonow/pola/auth/password"
)

type user struct {
	Username string
	Hash     string
}

type memUsers struct{ m map[string]*user }

func (s memUsers) FindByUsername(_ context.Context, name string) (*user, error) {
	if u, ok := s.m[name]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

// 32+ bytes: auth/jwt now rejects HS256 secrets weaker than 32 bytes.
var secret = []byte("test-secret-000000000000000000000000")

func newUsers(t *testing.T) memUsers {
	t.Helper()
	h, err := password.Hash("hunter2")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return memUsers{m: map[string]*user{"ada": {Username: "ada", Hash: h}}}
}

func bearerReq(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/me", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func TestJWTAuthenticator(t *testing.T) {
	users := newUsers(t)
	a := &JWTAuthenticator[user]{Users: users, Secret: secret}

	tok, err := IssueToken("ada", secret, time.Hour, nil)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	u, err := a.Authenticate(bearerReq(tok))
	if err != nil || u == nil || u.Username != "ada" {
		t.Fatalf("Authenticate = (%v,%v), want ada", u, err)
	}

	if _, err := a.Authenticate(bearerReq("")); !errors.Is(err, ErrNoCredentials) {
		t.Errorf("missing token err = %v, want ErrNoCredentials", err)
	}
	if _, err := a.Authenticate(bearerReq("garbage")); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("bad token err = %v, want ErrInvalidCredentials", err)
	}

	// A validly-signed token for an unknown subject is rejected.
	ghost, _ := IssueToken("nobody", secret, time.Hour, nil)
	if _, err := a.Authenticate(bearerReq(ghost)); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown subject err = %v, want ErrInvalidCredentials", err)
	}
}

func TestMiddlewareInjectsUser(t *testing.T) {
	users := newUsers(t)
	a := &JWTAuthenticator[user]{Users: users, Secret: secret}
	tok, _ := IssueToken("ada", secret, time.Hour, nil)

	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, ok := UserFromContext[user](r.Context()); ok {
			seen = u.Username
		}
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	Middleware(a).Wrap(next).ServeHTTP(rec, bearerReq(tok))
	if rec.Code != http.StatusOK || seen != "ada" {
		t.Errorf("authorized request: code %d, user %q; want 200/ada", rec.Code, seen)
	}
}

func TestMiddlewareRejectsMissing(t *testing.T) {
	a := &JWTAuthenticator[user]{Users: newUsers(t), Secret: secret}
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true })

	rec := httptest.NewRecorder()
	Middleware(a).Wrap(next).ServeHTTP(rec, bearerReq(""))
	if rec.Code != http.StatusUnauthorized || reached {
		t.Errorf("unauthenticated: code %d reached %v; want 401 and handler not reached", rec.Code, reached)
	}
}

func TestMiddlewareOptional(t *testing.T) {
	a := &JWTAuthenticator[user]{Users: newUsers(t), Secret: secret}
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		if _, ok := UserFromContext[user](r.Context()); ok {
			t.Error("no user should be present on an optional unauthenticated request")
		}
	})

	rec := httptest.NewRecorder()
	Middleware(a, WithOptional()).Wrap(next).ServeHTTP(rec, bearerReq(""))
	if !reached {
		t.Error("optional middleware should let an unauthenticated request through")
	}
}

func TestBasicAuthenticator(t *testing.T) {
	users := newUsers(t)
	a := &BasicAuthenticator[user]{Users: users, Password: func(u *user) string { return u.Hash }}

	good := httptest.NewRequest(http.MethodGet, "/", nil)
	good.SetBasicAuth("ada", "hunter2")
	if u, err := a.Authenticate(good); err != nil || u.Username != "ada" {
		t.Fatalf("valid basic = (%v,%v), want ada", u, err)
	}

	bad := httptest.NewRequest(http.MethodGet, "/", nil)
	bad.SetBasicAuth("ada", "wrong")
	if _, err := a.Authenticate(bad); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password err = %v, want ErrInvalidCredentials", err)
	}

	unknown := httptest.NewRequest(http.MethodGet, "/", nil)
	unknown.SetBasicAuth("ghost", "hunter2")
	if _, err := a.Authenticate(unknown); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown user err = %v, want ErrInvalidCredentials (no existence leak)", err)
	}
}

// nilUsers reports "not found" as (nil, nil), a common Go convention. The
// authenticators must reject that without dereferencing the nil user.
type nilUsers struct{}

func (nilUsers) FindByUsername(context.Context, string) (*user, error) { return nil, nil }

func TestAuthenticatorsHandleNilUser(t *testing.T) {
	basic := &BasicAuthenticator[user]{Users: nilUsers{}, Password: func(u *user) string { return u.Hash }}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("ada", "hunter2")
	if _, err := basic.Authenticate(req); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("basic nil user err = %v, want ErrInvalidCredentials", err)
	}

	jwt := &JWTAuthenticator[user]{Users: nilUsers{}, Secret: secret}
	tok, _ := IssueToken("ada", secret, time.Hour, nil)
	if u, err := jwt.Authenticate(bearerReq(tok)); u != nil || !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("jwt nil user = (%v,%v), want (nil, ErrInvalidCredentials)", u, err)
	}
}

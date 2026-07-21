package auth

import (
	"net/http"

	authjwt "github.com/polagonow/pola/auth/jwt"
	"github.com/polagonow/pola/auth/password"
)

// JWTAuthenticator validates a bearer token and loads the user named by one of
// its claims. It reuses the auth/jwt HS256 primitive for verification.
type JWTAuthenticator[T any] struct {
	// Users loads the account named by the token's subject claim.
	Users UserService[T]
	// Secret is the HS256 signing key (must match the one used to issue tokens).
	Secret []byte
	// ClaimName is the claim holding the username/subject. Defaults to "sub".
	ClaimName string
}

var _ Authenticator[struct{}] = (*JWTAuthenticator[struct{}])(nil)

// Authenticate verifies the request's bearer token and returns the user named by
// its subject claim.
func (a *JWTAuthenticator[T]) Authenticate(r *http.Request) (*T, error) {
	token := bearerToken(r)
	if token == "" {
		return nil, ErrNoCredentials
	}
	claims, err := authjwt.Verify(token, a.Secret)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	claim := a.ClaimName
	if claim == "" {
		claim = "sub"
	}
	subject, _ := claims[claim].(string)
	if subject == "" {
		return nil, ErrInvalidCredentials
	}
	u, err := a.Users.FindByUsername(r.Context(), subject)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// BasicAuthenticator validates HTTP Basic credentials against a bcrypt hash
// read from the loaded user. It reuses auth/password for the comparison.
type BasicAuthenticator[T any] struct {
	// Users loads the account named by the Basic username.
	Users UserService[T]
	// Password returns the stored bcrypt hash for a loaded user, e.g.
	// func(u *User) string { return u.PasswordHash }.
	Password func(*T) string
}

var _ Authenticator[struct{}] = (*BasicAuthenticator[struct{}])(nil)

// Authenticate reads the request's Basic credentials, loads the user and
// verifies the password. It returns [ErrInvalidCredentials] both for an unknown
// user and a wrong password, so the response can't be used to probe which
// usernames exist.
func (a *BasicAuthenticator[T]) Authenticate(r *http.Request) (*T, error) {
	username, pass, ok := r.BasicAuth()
	if !ok {
		return nil, ErrNoCredentials
	}
	u, err := a.Users.FindByUsername(r.Context(), username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	valid, err := password.Verify(pass, a.Password(u))
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

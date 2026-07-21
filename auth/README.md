# auth

Pluggable **API authentication**: verify a credential, load the current user,
expose it to the handler, and answer `401` when it's missing or invalid. Built on
the existing `auth/jwt` (HS256) and `auth/password` (bcrypt) primitives.

> Complements — doesn't replace — `middleware/requireauth`, which redirects
> unauthenticated **page** visitors to a sign-in screen. This package is for
> **API/action** auth with a `request.User`-style current user.

## Pieces

```go
type UserService[T any] interface {         // you implement this (or generate it)
    FindByUsername(ctx context.Context, username string) (*T, error)
}

type Authenticator[T any] interface {       // JWTAuthenticator / BasicAuthenticator / your own
    Authenticate(r *http.Request) (*T, error)
}

func Middleware[T any](a Authenticator[T], opts ...Option) core.Middleware
func UserFromContext[T any](ctx context.Context) (*T, bool)
func IssueToken(subject string, secret []byte, expiry time.Duration, extra map[string]any) (string, error)
```

## Usage

```go
// 1. Protect routes: authenticate every request, inject the user.
authenticator := &auth.JWTAuthenticator[User]{Users: users, Secret: secret}
reg.AddMiddleware(auth.Middleware(authenticator)) // or auth.Middleware(a, auth.WithOptional())

// 2. Read the current user in a handler.
func (c *Controller) Me(ctx core.Context) error {
    u, ok := auth.UserFromContext[User](ctx.Ctx())
    if !ok {
        return ctx.NoContent(http.StatusUnauthorized)
    }
    return ctx.JSON(http.StatusOK, u)
}

// 3. Log in: verify a password, then mint a token.
func (c *Controller) Login(ctx core.Context) error {
    var in dto.Credentials
    if err := ctx.Bind(&in); err != nil {
        return err
    }
    u, err := users.FindByUsername(ctx.Ctx(), in.Email)
    if err != nil {
        return ctx.NoContent(http.StatusUnauthorized)
    }
    if ok, _ := password.Verify(in.Password, u.PasswordHash); !ok {
        return ctx.NoContent(http.StatusUnauthorized)
    }
    tok, _ := auth.IssueToken(u.Email, secret, 24*time.Hour, nil)
    return ctx.JSON(http.StatusOK, core.M{"token": tok})
}
```

## Strategies

| Authenticator | Credential | Verification |
|---|---|---|
| `JWTAuthenticator[T]` | `Authorization: Bearer <token>` | `auth/jwt.Verify` (HS256), loads the user named by the `sub` claim (override with `ClaimName`) |
| `BasicAuthenticator[T]` | HTTP Basic | `auth/password.Verify` against the hash returned by the `Password` func |

Both return `ErrInvalidCredentials` for *unknown user* and *wrong credential*
alike, so responses can't be used to enumerate accounts. Implement
`Authenticator[T]` yourself for API keys, sessions, or third-party providers.

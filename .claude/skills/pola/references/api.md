# API features: auth, authorization, OpenAPI, CORS, health, per-route middleware

The "production API" layer added alongside DTOs/filtering. Real-world usage:
`examples/features-showcase` (filtering, DTOs, OpenAPI, health) and
`examples/saas-starter` (JWT sessions, sign-in/sign-up, protected dashboard).

## Authentication — `github.com/polagonow/pola/auth`

Generic over your user type. An `Authenticator[T]` turns a request into a user:

```go
type UserService[T any] interface {
    FindByUsername(ctx context.Context, username string) (*T, error)
}
type Authenticator[T any] interface {
    Authenticate(r *http.Request) (*T, error)
}
```

Built-ins: `auth.JWTAuthenticator[T]` (Bearer token; looks up the subject claim via
`Users`) and `auth.BasicAuthenticator[T]` (HTTP Basic; verifies via a password
getter). Passwords hash with `auth/password`; tokens sign with
`auth.IssueToken(subject string, secret []byte, expiry time.Duration, extra map[string]any)`.

Wire globally or per route:

```go
a := &auth.JWTAuthenticator[models.User]{Users: userSvc, Secret: secret}

// global: every request must authenticate (WithOptional() to allow anonymous)
pola.Use(core.PluginFunc{PluginName: "auth", Fn: func(r *core.Registry) {
    r.AddMiddleware(auth.Middleware(a))
}})

// per-route: implement Middleware() on the route struct in routes/
func (r *UserRoutes) Middleware() []core.RouteMiddleware {
    return []core.RouteMiddleware{auth.RouteMiddleware(r.auth)}
}

// in a handler / action / service:
user, ok := auth.UserFromContext[models.User](req.Context())
```

Options: `auth.WithOptional()` (continue unauthenticated), `auth.WithUnauthorized(fn)`
(custom 401 response). For tests: `auth.WithUser(ctx, u)` injects a user.

**Page-side auth** (cookie sessions + redirects) is configured in `Polafile.hcl`
instead: the `jwt` block (JWT cookie sessions) and the `protect` block (redirect
unauthenticated visitors of `paths` to `redirect`, default `/sign-in`) — see
`references/polafile.md`. `examples/saas-starter` combines both.

## Authorization — `github.com/polagonow/pola/auth/authz`

```go
authz.HasRole(user.Role, "owner")          // case-insensitive; also HasAny / HasAll
authz.RequireRole(user.Role, "owner")      // returns authz.ErrForbidden; also RequireAny

policy := authz.NewPolicy().
    Grant("owner", "team.manage", "billing.manage").
    Grant("member", "team.view")
policy.Can(user.Role, "billing.manage")
```

## OpenAPI — `github.com/polagonow/pola/openapi`

Generates an OpenAPI 3.0.3 document from the discovered `routes/` specs and serves
it with SwaggerUI:

```go
specs, _ := routes.Discover(reg)
doc := openapi.Generate(specs, openapi.Info{Title: "Shop API", Version: "1.0.0"})
reg.AddMiddleware(openapi.Serve(doc))       // /openapi.json + /openapi (UI)
// openapi.WithSpecPath("/spec.json"), openapi.WithUIPath("/docs-ui") to customize
```

Route structs add metadata by implementing `Meta()`:

```go
func (r *UserRoutes) Meta() map[string]any {
    return map[string]any{"summary": "Manage users", "tags": []string{"users"}}
}
```

Limitation: request/response body schemas are not yet inferred from DTOs.

## CORS — `github.com/polagonow/pola/middleware/cors`

```go
pola.Use(cors.Plugin(
    cors.WithAllowedOrigins("https://app.example.com"), // default: *
    cors.WithAllowCredentials(),                        // cookies/auth cross-origin
    cors.WithMaxAge(600),
))
// also: WithAllowedMethods, WithAllowedHeaders, WithExposedHeaders
```

Default (no options) is dev-permissive: any origin, common methods, reflected headers.

## Health checks — `github.com/polagonow/pola/middleware/health`

```go
pola.Use(health.Plugin(
    health.WithCheck("db", func(ctx context.Context) error { return db.PingContext(ctx) }),
))
```

`/healthz` (liveness) always returns 200; `/readyz` runs the checks and returns 200
or 503 with per-check status. Paths customizable via `WithLivePath` / `WithReadyPath`.

## Rate limiting / flash / i18n

- `middleware/ratelimit` — enable via the Polafile `rate_limit { requests_per_second, burst }` block.
- `flash` package + `flash { enabled = true }` block — one-shot messages in the session:
  `flash.Set(ctx, "notice", "Saved!")`, read with `flash.Get(ctx)`.
- `i18n` block + `github.com/polagonow/pola/i18n` — translation bundles loaded from
  `locales/` (`.json`/`.yaml`, filename = locale); locale detection middleware included.

## Request helpers — `github.com/polagonow/pola/request`

```go
id, err := request.PathParamInt(c, "id")     // also PathParam, PathParamUint
page   := request.QueryParamInt(c, "page", 1) // clamped; also QueryParam
```

## Route struct interfaces (routes/)

| Interface | Method | Effect |
|-----------|--------|--------|
| — | `GET/POST/PUT/PATCH/DELETE(w, r)` | HTTP handlers (path = directory) |
| `Middlewarer` | `Middleware() []core.RouteMiddleware` | Per-route middleware stack |
| `Metaer` | `Meta() map[string]any` | OpenAPI summary/tags |

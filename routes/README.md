# API Routes

Go-based API route handlers using livebud-style conventions. Route structs are auto-discovered and mounted to the server alongside frontend pages.

## Quick Start

Create a file at `routes/posts/route.go`:

```go
package posts

import (
    "encoding/json"
    "net/http"
)

type Route struct{}

func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{"posts": []string{}})
}

func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
    // handle create
}
```

That's it. No imports from `routes`, no `init()`, no `Register()`. The framework auto-generates registration via overlay at build time.

## CLI Scaffolding

```bash
# Create a route with default GET handler
pola generate route Posts

# Create with specific HTTP methods
pola generate route Posts --actions GET,POST,DELETE

# Create nested routes
pola generate route Posts/Comments --actions GET,POST
```

## HTTP Method Handlers

Define methods on your `Route` struct matching HTTP method names. All standard methods are supported:

```go
func (r *Route) GET(w http.ResponseWriter, req *http.Request)     {}
func (r *Route) POST(w http.ResponseWriter, req *http.Request)    {}
func (r *Route) PUT(w http.ResponseWriter, req *http.Request)     {}
func (r *Route) PATCH(w http.ResponseWriter, req *http.Request)   {}
func (r *Route) DELETE(w http.ResponseWriter, req *http.Request)  {}
func (r *Route) HEAD(w http.ResponseWriter, req *http.Request)    {}
func (r *Route) OPTIONS(w http.ResponseWriter, req *http.Request) {}
func (r *Route) CONNECT(w http.ResponseWriter, req *http.Request) {}
func (r *Route) TRACE(w http.ResponseWriter, req *http.Request)   {}
```

Each method must have the signature `func(http.ResponseWriter, *http.Request)`. The framework discovers them via reflection.

If a request matches a route pattern but not any registered method, a `405 Method Not Allowed` response is returned with an `Allow` header listing available methods.

## URL Pattern Derivation

URL patterns are derived from the Go package path using livebud's convention:

| Directory Structure | URL Pattern |
|---|---|
| `routes/health/` | `/health` |
| `routes/posts/` | `/posts` |
| `routes/users/` + `routes/users/posts/` | `/users/:id/posts` |
| `routes/users/` + `routes/users/posts/` + `routes/users/posts/comments/` | `/users/:user_id/posts/:id/comments` |

When a parent directory has a registered route and a child exists, the framework inserts a `:id` parameter between them (livebud's alternating-segment convention). Deeper nesting uses the singularized parent name: `:user_id`, `:post_id`, etc.

## Accessing URL Parameters

```go
import "github.com/polagonow/pola/routes"

func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
    // Get all params
    params := routes.Params(req) // map[string]any

    // Get a single param
    id := routes.Param(req, "id") // string
}
```

## Overriding Nesting Behavior

By default, if a parent directory has a `route.go`, the framework treats child routes as member routes (inserts `:id`). Override this with the `Member()` method:

```go
// routes/kampala/uganda/route.go
// Parent "kampala" has route.go, but we want /kampala/uganda (flat path)
type Route struct{}

func (*Route) Member() bool { return false }
```

Conversely, force member nesting even when the parent has no route:

```go
// routes/kampala/uganda/route.go
// Parent "kampala" has NO route.go, but we want /kampala/:id/uganda
type Route struct{}

func (*Route) Member() bool { return true }
```

Omit `Member()` entirely to use the default auto-detection.

## Custom Path Override

For full control over the URL pattern, implement `Path()`:

```go
type Route struct{}

func (*Route) Path() string { return "/api/v2/custom-endpoint" }
```

The path must start with `/`.

## Dependency Injection

Route structs can receive dependencies from the DI container by implementing `Init`:

```go
import "github.com/samber/do/v2"

type Route struct {
    db *sql.DB
}

func (r *Route) Init(injector do.Injector) error {
    r.db = do.MustInvoke[*sql.DB](injector)
    return nil
}

func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
    // use r.db
}
```

`Init` is called once during `Build()`, before any requests are served.

## Page Priority

API routes and frontend pages share the same URL namespace. The resolution order is:

1. **GET requests with a matching page** — the page wins.
2. **Non-GET requests** (POST, PUT, DELETE, etc.) — API route handles it.
3. **GET requests with no matching page** — API route handles it.

This means you can have both `app/posts/page.tsx` (frontend) and `routes/posts/route.go` (API) at `/posts`. The page serves GET, the route serves POST/PUT/DELETE.

## How Auto-Registration Works

You don't need to write any registration code. The framework:

1. Discovers all packages under `routes/` with `.go` files
2. Generates a `pola_route_init.go` overlay file per package containing `routes.Register(&Route{})`
3. Blank-imports each route package in the generated `pola_plugins.go`
4. At startup, `Router.Build()` drains all registered structs and compiles the routing table

This is the same overlay pattern used for plugin registration and action bridges.

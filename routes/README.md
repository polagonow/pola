# API Routes

Go-based API route handlers. Routes are auto-discovered and mounted to the
server alongside frontend pages.

Handlers are written against the framework-neutral `core.Context`, so the
selected web framework (`std` / `gin` / `echo` / `chi`) can be swapped without
touching route code. See [`webframework/README.md`](../webframework/README.md)
for framework selection.

## Quick Start

A route is either a **struct** with verb methods (use when it needs injected
dependencies) or a package-level **function** named after an HTTP verb (use for
simple, dependency-free handlers).

Create a file at `routes/posts/route.go`:

```go
package posts

import "github.com/polagonow/pola/core"

type Route struct{}

func (r *Route) GET(c core.Context) error {
    return c.JSON(200, core.M{"posts": []string{}})
}

func (r *Route) POST(c core.Context) error {
    // handle create
    return c.NoContent(201)
}
```

Or, for a handler with no dependencies, a function-based route:

```go
package posts

import "github.com/polagonow/pola/core"

// GET /posts
func GET(c core.Context) error {
    return c.JSON(200, core.M{"posts": []string{}})
}
```

That's it. No imports from `routes`, no `init()`, no `Register()`. The framework
auto-generates registration via an overlay at build time.

## CLI Scaffolding

```bash
# Create a route with a default GET handler (struct-based)
pola generate route Posts

# Create with specific HTTP methods (positional, space- or comma-separated)
pola generate route Posts GET,POST,DELETE

# Create nested routes
pola generate route Posts/Comments GET POST

# Create a function-based route
pola generate route Ping --func

# Wire a route to a generated service via DI
pola generate route Posts GET,POST,DELETE --service=Post
```

If no methods are provided, the generator defaults to `GET`.

## HTTP Method Handlers

Define methods on your `Route` struct (or package-level functions) matching HTTP
method names. All standard methods are supported:

```go
func (r *Route) GET(c core.Context) error     { return nil }
func (r *Route) POST(c core.Context) error    { return nil }
func (r *Route) PUT(c core.Context) error     { return nil }
func (r *Route) PATCH(c core.Context) error   { return nil }
func (r *Route) DELETE(c core.Context) error  { return nil }
func (r *Route) HEAD(c core.Context) error    { return nil }
func (r *Route) OPTIONS(c core.Context) error { return nil }
func (r *Route) CONNECT(c core.Context) error { return nil }
func (r *Route) TRACE(c core.Context) error   { return nil }
```

Each handler must have the signature `func(core.Context) error`. The framework
discovers them via reflection.

Returning a non-nil error lets the adapter centralize error handling: if the
handler has not already written a response, the error is translated into a
`500`. Otherwise write the response yourself with `c.JSON`, `c.String`,
`c.NoContent`, etc.

If a request matches a route pattern but not any registered method, a
`405 Method Not Allowed` response is returned with an `Allow` header listing the
available methods.

## The Context

`core.Context` is the framework-neutral request/response abstraction. Common
methods:

```go
c.Request()                 // *http.Request
c.Writer()                  // http.ResponseWriter
c.Ctx()                     // context.Context (shorthand for Request().Context())

c.Param("id")               // path parameter
c.Query("page")             // query-string value
c.Header("Accept")          // request header

// Binding has two families — ShouldBind* return the error; Bind* also write a
// 400 JSON response on failure (gin's "must" semantics).
c.ShouldBind(&v)            // JSON body   -> struct                (must: c.Bind)
c.ShouldBindUri(&v)         // path params -> struct `uri:"id"`     (must: c.BindUri)
c.ShouldBindQuery(&v)       // query       -> struct `query:"page"` (must: c.BindQuery)
c.ShouldBindHeader(&v)      // headers     -> struct `header:"X-T"` (must: c.BindHeader)
c.ShouldBindForm(&v)        // form body   -> struct `form:"email"` (must: c.BindForm)
c.FormValue("name")         // form field
c.FormFile("avatar")        // *multipart.FileHeader
c.MultipartForm()           // full multipart form

c.JSON(200, v)              // write JSON response
c.String(200, "ok")         // write text/plain response
c.NoContent(204)            // write status with empty body
c.SetHeader(k, v)           // set a response header
c.Redirect(302, "/login")   // HTTP redirect
c.Status()                  // status code written so far (0 if none)
```

`core.M` is a convenience alias for `map[string]any`, e.g.
`c.JSON(200, core.M{"status": "ok"})`.

## Accessing URL Parameters

```go
func (r *Route) GET(c core.Context) error {
    id := c.Param("id")
    return c.JSON(200, core.M{"id": id})
}
```

Params are also mirrored onto the request context, so helpers and legacy
`net/http`-style code can read them via `core.Param`:

```go
import "github.com/polagonow/pola/core"

func (r *Route) GET(c core.Context) error {
    id := core.Param(c.Request(), "id") // string
    _ = core.Params(c.Request())        // map[string]any (all params)
    return c.JSON(200, core.M{"id": id})
}
```

## Binding into Structs

Beyond reading values one at a time, you can bind a whole request source into a
typed struct. Each source has its own struct tag and — mirroring gin — comes in
two families:

- **`ShouldBindX`** decode into the struct and **return the error** for you to
  handle.
- **`BindX`** (the "must" variants) do the same but, on failure, **also write a
  400 JSON response** — so the handler can just `return err`.

| Should (returns error) | Must (writes 400) | Tag | Source |
|---|---|---|---|
| `c.ShouldBind(&v)` | `c.Bind(&v)` | — | JSON body |
| `c.ShouldBindUri(&v)` | `c.BindUri(&v)` | `uri:"id"` | URL path parameters |
| `c.ShouldBindQuery(&v)` | `c.BindQuery(&v)` | `query:"page"` | URL query string |
| `c.ShouldBindHeader(&v)` | `c.BindHeader(&v)` | `header:"X-Token"` | request headers (case-insensitive) |
| `c.ShouldBindForm(&v)` | `c.BindForm(&v)` | `form:"email"` | POST form body (urlencoded or multipart) |

Neither family validates: missing fields are left at their zero value, so run
`validation.Validate(&v)` afterward using `validate:"..."` tags. A single struct
can carry several binding tags plus `validate`:

```go
import (
    "github.com/polagonow/pola/core"
    "github.com/polagonow/pola/validation"
)

type listInput struct {
    ID    int      `uri:"id"             validate:"required"`
    Page  int      `query:"page"`
    Tags  []string `query:"tag"`         // repeated ?tag=a&tag=b -> []string
    Token string   `header:"X-Api-Token" validate:"required"`
}

// ShouldBind* style: you choose the status and body for every failure.
func (r *Route) GET(c core.Context) error {
    var in listInput
    if err := c.ShouldBindUri(&in); err != nil {
        return c.JSON(400, core.M{"error": err.Error()})
    }
    if err := c.ShouldBindQuery(&in); err != nil {
        return c.JSON(400, core.M{"error": err.Error()})
    }
    if err := c.ShouldBindHeader(&in); err != nil {
        return c.JSON(400, core.M{"error": err.Error()})
    }
    if err := validation.Validate(&in); err != nil {
        return c.JSON(422, core.M{"error": err.Error()})
    }
    return c.JSON(200, in)
}
```

With the **must** variants a bind failure writes the 400 for you, so the handler
shrinks to `return err`:

```go
func (r *Route) GET(c core.Context) error {
    var in listInput
    if err := c.BindUri(&in); err != nil {
        return err // 400 already written
    }
    if err := c.BindQuery(&in); err != nil {
        return err
    }
    if err := validation.Validate(&in); err != nil {
        return c.JSON(422, core.M{"error": err.Error()})
    }
    return c.JSON(200, in)
}
```

Notes:

- **Missing vs zero.** An absent value leaves the field at its zero value, so a
  numeric field can't tell "absent" from `0`. Use `validate:"required"` to reject
  missing values, or a pointer field (`*int`) when `0`/`""` is legitimate — it
  stays `nil` when absent.
- **Slices.** Repeated query/header/form values bind to slice fields
  (`[]string`, `[]int`, ...). Path params are single-valued.
- **Types.** `string`, `bool`, the sized `int*`/`uint*`, `float32/64`, pointers
  to those, and any type implementing `encoding.TextUnmarshaler` are supported.
  Untagged and `-`-tagged fields are skipped (no field-name fallback).
- **`query` vs `form`.** Pola keeps these tags distinct (gin overloads `form`
  for both): `query:` reads the URL query string, `form:` reads the request body.

## URL Pattern Derivation

URL patterns are derived from the Go package path using livebud's convention:

| Directory Structure | URL Pattern |
|---|---|
| `routes/health/` | `/health` |
| `routes/posts/` | `/posts` |
| `routes/users/` + `routes/users/posts/` | `/users/:id/posts` |
| `routes/users/` + `routes/users/posts/` + `routes/users/posts/comments/` | `/users/:user_id/posts/:id/comments` |

When a parent directory has a registered route and a child exists, the framework
inserts a `:id` parameter between them (livebud's alternating-segment
convention). Deeper nesting uses the singularized parent name: `:user_id`,
`:post_id`, etc.

`GET`/`POST` map to the collection (the base pattern); `PUT`/`PATCH`/`DELETE`
map to the member (base + `/:id`). Patterns use Pola's `:param` / `:...rest`
syntax; each adapter translates them to its native syntax (see
`webframework/pattern`).

## Overriding Nesting Behavior

By default, if a parent directory has a `route.go`, the framework treats child
routes as member routes (inserts `:id`). Override this with the `Member()`
method:

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

Struct-based routes receive the DI registry through a `NewRoute(r *core.Registry)`
constructor and pull whatever dependencies they need from it with
`core.MustInvoke[T](r)`. The build-time overlay calls the constructor for you
before any requests are served:

```go
package posts

import (
    "github.com/polagonow/pola/core"
    "myapp/services"
)

type Route struct {
    svc *services.PostService
}

// The generated overlay invokes NewRoute with the DI registry; the constructor
// pulls its own dependencies out of it. Add or remove fields freely — no need
// to touch a separate wiring layer.
func NewRoute(r *core.Registry) *Route {
    return &Route{svc: core.MustInvoke[*services.PostService](r)}
}

func (r *Route) GET(c core.Context) error {
    posts, err := r.svc.List(c.Ctx())
    if err != nil {
        return c.JSON(500, core.M{"error": err.Error()})
    }
    return c.JSON(200, posts)
}
```

`pola generate route Posts --service=Post` scaffolds this registry-style wiring.
Function-based routes have no constructor, so `--func` cannot be combined with
`--service`.

**Backward-compatible explicit-dependency style also works.** If you prefer, list
the dependencies as constructor parameters and the autoloader will resolve them
by type — service interfaces, `storage.Storage`, and blob repositories are all
picked up:

```go
func NewRoute(svc services.PostServiceInterface) *Route {
    return &Route{svc: svc}
}
```

Mixed signatures (`NewRoute(r *core.Registry, svc services.PostServiceInterface)`)
are supported as well; the registry parameter is passed through verbatim while
the other parameters are resolved with `core.Invoke[T]`.

## File Uploads

`core.Context` exposes gin/echo-style upload helpers backed by every adapter:

```go
func (r *Route) POST(c core.Context) error {
    fh, err := c.FormFile("avatar") // *multipart.FileHeader
    if err != nil {
        return c.JSON(400, core.M{"error": "avatar required"})
    }
    f, err := fh.Open()
    if err != nil {
        return c.JSON(400, core.M{"error": err.Error()})
    }
    defer f.Close()
    // ... save f via storage.Storage ...
    return c.NoContent(204)
}
```

For multipart payloads that combine a JSON blob with file parts, use
`request.DecodeMultipartJSON(c, &v, maxMemory)` to decode the `data` field, then
read files with `c.FormFile`.

## Page Priority

API routes and frontend pages share the same URL namespace. The resolution
order is:

1. **GET requests with a matching page** — the page wins.
2. **Non-GET requests** (POST, PUT, DELETE, etc.) — API route handles it.
3. **GET requests with no matching page** — API route handles it.

This means you can have both `app/posts/page.tsx` (frontend) and
`routes/posts/route.go` (API) at `/posts`. The page serves GET, the route serves
POST/PUT/DELETE.

## How Auto-Registration Works

You don't need to write any registration code. The framework:

1. Discovers all packages under `routes/` that declare a `Route` struct or
   verb-named functions (directories with neither — e.g. shared helpers — are
   skipped).
2. Generates a `pola_route_init.go` overlay file per package. For struct routes
   it emits `routes.Register(func(r *core.Registry) any { ... })` (calling
   `NewRoute` with resolved dependencies); for function routes it emits
   `routes.RegisterFunc("GET", GET)`.
3. Blank-imports each route package in the generated `pola_plugins.go`.
4. At build, `routes.Discover` drains all registrations, derives the RESTful URL
   patterns, and returns `RouteSpecs`. The pipeline registers each spec onto the
   selected `core.WebFramework`, which owns matching and dispatch.

This is the same overlay pattern used for plugin registration and action
bridges.

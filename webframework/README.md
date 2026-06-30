# webframework

Pluggable HTTP web frameworks for Pola.

Pola's HTTP layer is an interface, `core.WebFramework`. The selected adapter's
engine becomes the application's root `http.Handler`; the pipeline mounts Pola's
machinery (API routes, page renderer, reserved `/_pola/*` endpoints, server
actions, middleware) onto it.

Adapters:

| Name   | Backend                          | Extra dependency            |
| ------ | -------------------------------- | --------------------------- |
| `std`  | Go `net/http` (default)          | none                        |
| `gin`  | github.com/gin-gonic/gin         | gin                         |
| `echo` | github.com/labstack/echo/v4      | echo                        |
| `chi`  | github.com/go-chi/chi/v5         | chi (already a transitive)  |

All four are `net/http`-compatible. Only the **selected** adapter is imported
into a built app, so unused frameworks add no dependencies. `std` is the default
and pulls nothing extra.

## Selecting a framework

Resolution order: CLI flag > `POLA_WEB_FRAMEWORK` env var > Polafile > `"std"`.

```hcl
# Polafile.hcl
pola {
  framework = "gin"   # std | gin | echo | chi
}
```

```bash
pola new myapp --framework gin
pola serve --framework echo
POLA_WEB_FRAMEWORK=chi pola build
```

## Writing handlers

Route handlers are written against the framework-neutral `core.Context`, never a
concrete `*gin.Context`. Switching frameworks requires **no** changes to route
code.

```go
package users

import "github.com/polagonow/pola/core"

// Struct-based route (gets dependencies via NewRoute + DI).
type Route struct{ svc *services.UserService }

func NewRoute(svc *services.UserService) *Route { return &Route{svc} }

// GET /users
func (r *Route) GET(c core.Context) error {
	users, err := r.svc.List(c.Ctx())
	if err != nil {
		return c.JSON(500, core.M{"error": err.Error()})
	}
	return c.JSON(200, users)
}

// POST /users
func (r *Route) POST(c core.Context) error {
	var in CreateUser
	if err := c.Bind(&in); err != nil {
		return c.JSON(400, core.M{"error": "invalid body"})
	}
	u, err := r.svc.Create(c.Ctx(), in)
	if err != nil {
		return c.JSON(500, core.M{"error": err.Error()})
	}
	return c.JSON(201, u)
}
```

### Function-based routes

For simple, dependency-free handlers, declare package-level functions named
after HTTP verbs instead of a struct:

```go
package health

import (
	"runtime"

	"github.com/polagonow/pola/core"
)

// GET /health
func GET(c core.Context) error {
	return c.JSON(200, core.M{"status": "ok", "go": runtime.Version()})
}
```

Scaffold one with `pola generate route Ping --func`.

### File uploads

`core.Context` exposes gin/echo-style upload helpers backed by every adapter:

```go
func (r *Route) POST(c core.Context) error {
	fh, err := c.FormFile("avatar")        // *multipart.FileHeader
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

## Conventions (shared across frameworks)

- URL patterns are derived from the route package path (RESTful):
  `routes/users/posts` → `/users/:id/posts`. Override with `Path() string`.
- `GET`/`POST` map to the collection; `PUT`/`PATCH`/`DELETE` map to the member
  (`/:id`). Patterns use Pola's `:param` / `:...rest` syntax; each adapter
  translates to its native syntax (see `webframework/pattern`).
- Frontend pages win for `GET` on a shared path; API routes handle non-GET and
  GET paths with no page.
- Pola middleware (`core.Middleware`) is `net/http`-based and wraps the whole
  engine, so it works unchanged on every adapter.

## Adding a new adapter

Implement `core.WebFramework` and a `core.Context` over the framework's native
types, then expose `Plugin()` providing a `core.WebFrameworkFactory`. See
`std/` for the reference implementation and `conformance_test.go` for the suite
every adapter must pass.

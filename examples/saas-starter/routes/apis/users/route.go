package users

import (
	"net/http"

	"github.com/polagonow/pola/core"

	"saas-starter/lib/session"
	"saas-starter/repositories"
)

// Route serves GET /api/user — the current authenticated user, or null. Consumed
// by the dashboard's useUser() SWR hook.
type Route struct {
	users repositories.UserRepository
}

// NewRoute resolves the Route's dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{users: core.MustInvoke[repositories.UserRepository](r)}
}

// Path pins the exact URL (overriding the directory-derived default).
func (r *Route) Path() string { return "/api/user" }

// GET /api/user
func (r *Route) GET(c core.Context) error {
	uid, ok := session.UserID(c.Ctx())
	if !ok {
		return c.JSON(http.StatusOK, nil)
	}
	u, err := r.users.Get(c.Ctx(), uid)
	if err != nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, u) // PasswordHash is hidden via json:"-"
}

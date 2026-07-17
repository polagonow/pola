package activity

import (
	"net/http"

	"github.com/polagonow/pola/core"

	"saas-starter/lib/session"
	"saas-starter/repositories"
)

// Route serves GET /api/activity — the current user's recent activity log
// entries (newest first). Consumed by the dashboard activity page via SWR.
type Route struct {
	activity repositories.ActivityLogRepository
}

// NewRoute resolves the Route's dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{activity: core.MustInvoke[repositories.ActivityLogRepository](r)}
}

// Path pins the exact URL.
func (r *Route) Path() string { return "/api/activity" }

// GET /api/activity
func (r *Route) GET(c core.Context) error {
	uid, ok := session.UserID(c.Ctx())
	if !ok {
		return c.JSON(http.StatusOK, []any{})
	}
	logs, err := r.activity.ListForUser(c.Ctx(), uid, 20)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, logs)
}

package teams

import (
	"net/http"

	"github.com/polagonow/pola/core"

	"saas-starter/db/models"
	"saas-starter/lib/session"
	"saas-starter/repositories"
)

// Route serves GET /api/team — the current user's team plus its members.
// Consumed by the dashboard's team SWR hook.
type Route struct {
	teams   repositories.TeamRepository
	members repositories.TeamMemberRepository
	users   repositories.UserRepository
}

// NewRoute resolves the Route's dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{
		teams:   core.MustInvoke[repositories.TeamRepository](r),
		members: core.MustInvoke[repositories.TeamMemberRepository](r),
		users:   core.MustInvoke[repositories.UserRepository](r),
	}
}

// Path pins the exact URL.
func (r *Route) Path() string { return "/api/team" }

type memberView struct {
	ID     uint   `json:"id"`
	UserID uint   `json:"userId"`
	Role   string `json:"role"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type teamView struct {
	*models.Team
	Members []memberView `json:"members"`
}

// GET /api/team
func (r *Route) GET(c core.Context) error {
	uid, ok := session.UserID(c.Ctx())
	if !ok {
		return c.JSON(http.StatusOK, nil)
	}
	team, err := r.teams.GetForUser(c.Ctx(), uid)
	if err != nil {
		return c.JSON(http.StatusOK, nil)
	}
	members, _ := r.members.ListByTeam(c.Ctx(), team.ID)
	views := make([]memberView, 0, len(members))
	for _, m := range members {
		mv := memberView{ID: m.ID, UserID: m.UserID, Role: m.Role}
		if u, err := r.users.Get(c.Ctx(), m.UserID); err == nil {
			mv.Name = u.Name
			mv.Email = u.Email
		}
		views = append(views, mv)
	}
	return c.JSON(http.StatusOK, teamView{Team: team, Members: views})
}

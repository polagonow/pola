package register

import (
	"net/http"

	"beego-features-demo/services"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/flash"
	"github.com/polagonow/pola/i18n"
	"github.com/polagonow/pola/middleware/session"
)

type Route struct {
	svc *services.UserService
}

func NewRoute(svc *services.UserService) *Route {
	return &Route{svc: svc}
}

func (r *Route) POST(c core.Context) error {
	var input struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if err := c.ShouldBind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": "invalid JSON"})
	}

	user, err := r.svc.Register(c.Ctx(), input.Username, input.Password, input.DisplayName)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, core.M{"error": err.Error()})
	}

	session.Set(c.Ctx(), "user_id", user.ID)
	session.Set(c.Ctx(), "username", user.Username)
	flash.Set(c.Ctx(), "success", i18n.T(c.Ctx(), "register_success"))

	return c.JSON(http.StatusCreated, core.M{
		"id":       user.ID,
		"username": user.Username,
	})
}

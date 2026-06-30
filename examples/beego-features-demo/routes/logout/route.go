package logout

import (
	"net/http"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/flash"
	"github.com/polagonow/pola/i18n"
	"github.com/polagonow/pola/middleware/session"
)

type Route struct{}

func (r *Route) POST(c core.Context) error {
	session.Clear(c.Ctx())
	flash.Set(c.Ctx(), "success", i18n.T(c.Ctx(), "logout_success"))

	return c.JSON(http.StatusOK, core.M{"status": "ok"})
}

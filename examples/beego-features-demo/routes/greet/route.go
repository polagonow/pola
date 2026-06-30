package greet

import (
	"net/http"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/i18n"
)

type Route struct{}

func (r *Route) GET(c core.Context) error {
	name := c.Query("name")
	if name == "" {
		name = "World"
	}

	return c.JSON(http.StatusOK, core.M{
		"locale":   i18n.Locale(c.Ctx()),
		"greeting": i18n.T(c.Ctx(), "greeting", name),
	})
}

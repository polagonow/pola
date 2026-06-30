package contacts

import (
	"net/http"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/validation"
)

type Route struct{}

type Contact struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone" validate:"omitempty,e164"`
}

var v = validation.New()

func (r *Route) POST(c core.Context) error {
	var input Contact
	if err := c.ShouldBind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": "invalid JSON"})
	}

	if err := v.Validate(&input); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, core.M{"errors": err})
	}

	return c.JSON(http.StatusCreated, core.M{
		"status":  "ok",
		"contact": input,
	})
}

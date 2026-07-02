package greetings

import (
	"net/http"

	"mcp-hello/repositories"
	"mcp-hello/services"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/request"
	"github.com/polagonow/pola/validation"
)

// Route handles /greetings requests.
type Route struct {
	svc *services.GreetingService
}

// NewRoute creates a Route with its service dependency.
func NewRoute(svc *services.GreetingService) *Route {
	return &Route{svc: svc}
}

// GET /greetings
func (r *Route) GET(c core.Context) error {
	id := c.Param("id")
	if id != "" {
		n, err := request.PathParamUint(c, "id")
		if err != nil {
			return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
		}
		entity, err := r.svc.Get(c.Ctx(), n)
		if err != nil {
			return c.JSON(http.StatusNotFound, core.M{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, entity)
	}

	params := repository.ListParams{
		Page:    request.QueryParamInt(c, "page", 1),
		PerPage: request.QueryParamInt(c, "per_page", repository.DefaultPerPage),
	}
	result, err := r.svc.List(c.Ctx(), params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// POST /greetings
func (r *Route) POST(c core.Context) error {
	var entity repositories.Greeting
	if err := c.ShouldBind(&entity); err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	if err := validation.Validate(&entity); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, core.M{"error": err.Error()})
	}
	if err := r.svc.Create(c.Ctx(), &entity); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, entity)
}

// PUT /greetings
func (r *Route) PUT(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.Greeting
	if err := c.ShouldBind(&entity); err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	if err := validation.Validate(&entity); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, core.M{"error": err.Error()})
	}
	entity.ID = id
	if err := r.svc.Update(c.Ctx(), &entity); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, entity)
}

// PATCH /greetings
func (r *Route) PATCH(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.Greeting
	if err := c.ShouldBind(&entity); err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	if err := validation.Validate(&entity); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, core.M{"error": err.Error()})
	}
	entity.ID = id
	if err := r.svc.Update(c.Ctx(), &entity); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, entity)
}

// DELETE /greetings
func (r *Route) DELETE(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	if err := r.svc.Delete(c.Ctx(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

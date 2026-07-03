package todos

import (
	"net/http"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/request"
	"github.com/polagonow/pola/validation"

	"todo-api/repositories"
	"todo-api/services"
)

// Route handles /todos requests.
type Route struct {
	svc services.TodoServiceInterface
}

// NewRoute creates a Route, resolving its dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{svc: core.MustInvoke[services.TodoServiceInterface](r)}
}

// GET /todos
func (r *Route) GET(c core.Context) error {
	if request.PathParam(c, "id") != "" {
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

	params := repositories.ListParams{
		Page:    request.QueryParamInt(c, "page", 1),
		PerPage: request.QueryParamInt(c, "per_page", repositories.DefaultPerPage),
	}
	result, err := r.svc.List(c.Ctx(), params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// POST /todos
func (r *Route) POST(c core.Context) error {
	var entity repositories.Todo
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

// PUT /todos
func (r *Route) PUT(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.Todo
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

// PATCH /todos
func (r *Route) PATCH(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.Todo
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

// DELETE /todos
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

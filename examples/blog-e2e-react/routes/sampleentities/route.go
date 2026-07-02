package sampleentities

import (
	"net/http"

	"blog-e2e-react/repositories"
	"blog-e2e-react/services"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/request"
	"github.com/polagonow/pola/validation"
)

// Route handles /sampleentities requests.
type Route struct {
	svc *services.SampleEntityService
}

// NewRoute creates a Route with its service dependency.
func NewRoute(svc *services.SampleEntityService) *Route {
	return &Route{svc: svc}
}

// GET /sampleentities
func (r *Route) GET(c core.Context) error {
	if c.Param("id") != "" {
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

// POST /sampleentities
func (r *Route) POST(c core.Context) error {
	var entity repositories.SampleEntity
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

// PUT /sampleentities
func (r *Route) PUT(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.SampleEntity
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

// PATCH /sampleentities
func (r *Route) PATCH(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.SampleEntity
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

// DELETE /sampleentities
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

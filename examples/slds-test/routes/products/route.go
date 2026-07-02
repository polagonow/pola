package products

import (
	"net/http"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/request"
	"github.com/polagonow/pola/validation"
	"slds-test/repositories"
	"slds-test/services"
)

// Route handles /products requests.
type Route struct {
	svc *services.ProductService
}

// NewRoute creates a Route with its service dependency.
func NewRoute(svc *services.ProductService) *Route {
	return &Route{svc: svc}
}

// GET /products
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

// POST /products
func (r *Route) POST(c core.Context) error {
	var entity repositories.Product
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

// PUT /products
func (r *Route) PUT(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.Product
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

// PATCH /products
func (r *Route) PATCH(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	var entity repositories.Product
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

// DELETE /products
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

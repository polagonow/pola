package products

import (
	"net/http"

	"features-showcase/db/models"
	dto "features-showcase/dto"
	"features-showcase/services"
	"github.com/polagonow/pola/core"
	poladto "github.com/polagonow/pola/dto"
	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/request"
)

// Route handles /products requests. Handlers speak DTOs at the boundary:
// requests bind Create/Update DTOs mapped onto the model, and responses convert
// the model to a response DTO (so internal-only columns never leak).
type Route struct {
	svc services.ProductServiceInterface
}

// NewRoute creates a Route, resolving its dependencies from the DI registry.
func NewRoute(r *core.Registry) *Route {
	return &Route{svc: core.MustInvoke[services.ProductServiceInterface](r)}
}

// GET /products
func (r *Route) GET(c core.Context) error {
	if request.PathParam(c, "id") != "" {
		n, err := request.PathParamUint(c, "id")
		if err != nil {
			return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
		}
		entity, err := r.svc.Get(c.Ctx(), n)
		if err != nil {
			if repository.IsNotFound(err) {
				return c.JSON(http.StatusNotFound, core.M{"error": err.Error()})
			}
			return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, poladto.MustConvert[dto.Product](entity))
	}

	// Showcase — query-driven filtering/sorting/field-select straight from the
	// query string (repository.ParseListQuery), e.g.
	//   ?filter=name||$cont||wid&filter=price||$gte||10&sort=price,desc&per_page=5
	params := repository.ParseListQuery(c.Request().URL.Query())
	result, err := r.svc.List(c.Ctx(), params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	items := make([]dto.Product, len(result.Items))
	for i, it := range result.Items {
		items[i] = poladto.MustConvert[dto.Product](it)
	}
	return c.JSON(http.StatusOK, core.M{
		"items":       items,
		"total":       result.Total,
		"page":        result.Page,
		"per_page":    result.PerPage,
		"total_pages": result.TotalPages,
	})
}

// POST /products
func (r *Route) POST(c core.Context) error {
	var in dto.CreateProduct
	if err := c.Bind(&in); err != nil {
		return err
	}
	var entity models.Product
	poladto.Copy(&entity, in)
	if err := r.svc.Create(c.Ctx(), &entity); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, poladto.MustConvert[dto.Product](&entity))
}

// PUT /products
func (r *Route) PUT(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	entity, err := r.svc.Get(c.Ctx(), id)
	if err != nil {
		if repository.IsNotFound(err) {
			return c.JSON(http.StatusNotFound, core.M{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}

	var in dto.UpdateProduct
	if err := c.Bind(&in); err != nil {
		return err
	}
	poladto.Copy(entity, in)
	if err := r.svc.Update(c.Ctx(), entity); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, poladto.MustConvert[dto.Product](entity))
}

// PATCH /products
func (r *Route) PATCH(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	entity, err := r.svc.Get(c.Ctx(), id)
	if err != nil {
		if repository.IsNotFound(err) {
			return c.JSON(http.StatusNotFound, core.M{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}

	var in dto.UpdateProduct
	if err := c.Bind(&in); err != nil {
		return err
	}
	poladto.Copy(entity, in)
	if err := r.svc.Update(c.Ctx(), entity); err != nil {
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, poladto.MustConvert[dto.Product](entity))
}

// DELETE /products
func (r *Route) DELETE(c core.Context) error {
	id, err := request.PathParamUint(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, core.M{"error": err.Error()})
	}
	if err := r.svc.Delete(c.Ctx(), id); err != nil {
		if repository.IsNotFound(err) {
			return c.JSON(http.StatusNotFound, core.M{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, core.M{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

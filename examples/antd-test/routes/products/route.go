package products

import (
	"net/http"

	"antd-test/repositories"
	"antd-test/services"
	"github.com/polagonow/pola/routes"
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
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
	id := routes.Param(req, "id")
	if id != "" {
		n, err := routes.ParseUintID(req)
		if err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		entity, err := r.svc.Get(req.Context(), n)
		if err != nil {
			routes.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		routes.WriteJSON(w, http.StatusOK, entity)
		return
	}

	params := repositories.ListParams{
		Page:    routes.QueryInt(req, "page", 1),
		PerPage: routes.QueryInt(req, "per_page", repositories.DefaultPerPage),
	}
	result, err := r.svc.List(req.Context(), params)
	if err != nil {
		routes.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	routes.WriteJSON(w, http.StatusOK, result)
}

// POST /products
func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	var entity repositories.Product
	if err := routes.DecodeJSON(req, &entity); err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := routes.Validate(&entity); err != nil {
		routes.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := r.svc.Create(req.Context(), &entity); err != nil {
		routes.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	routes.WriteJSON(w, http.StatusCreated, entity)
}

// PUT /products
func (r *Route) PUT(w http.ResponseWriter, req *http.Request) {
	id, err := routes.ParseUintID(req)
	if err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var entity repositories.Product
	if err := routes.DecodeJSON(req, &entity); err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := routes.Validate(&entity); err != nil {
		routes.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	entity.ID = id
	if err := r.svc.Update(req.Context(), &entity); err != nil {
		routes.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	routes.WriteJSON(w, http.StatusOK, entity)
}

// PATCH /products
func (r *Route) PATCH(w http.ResponseWriter, req *http.Request) {
	id, err := routes.ParseUintID(req)
	if err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var entity repositories.Product
	if err := routes.DecodeJSON(req, &entity); err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := routes.Validate(&entity); err != nil {
		routes.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	entity.ID = id
	if err := r.svc.Update(req.Context(), &entity); err != nil {
		routes.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	routes.WriteJSON(w, http.StatusOK, entity)
}

// DELETE /products
func (r *Route) DELETE(w http.ResponseWriter, req *http.Request) {
	id, err := routes.ParseUintID(req)
	if err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := r.svc.Delete(req.Context(), id); err != nil {
		routes.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

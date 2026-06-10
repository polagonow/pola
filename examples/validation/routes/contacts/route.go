package contacts

import (
	"net/http"

	"validation/repositories"
	"validation/services"

	"github.com/polagonow/pola/routes"
)

// Route handles /contacts requests with govalidator-backed validation.
type Route struct {
	svc *services.ContactService
}

// NewRoute creates a Route with its service dependency.
func NewRoute(svc *services.ContactService) *Route {
	return &Route{svc: svc}
}

// GET /contacts
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

// POST /contacts — validates email, url, alpha, and numeric fields.
func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	var entity repositories.Contact
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

// PUT /contacts/:id
func (r *Route) PUT(w http.ResponseWriter, req *http.Request) {
	id, err := routes.ParseUintID(req)
	if err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var entity repositories.Contact
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

// DELETE /contacts/:id
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

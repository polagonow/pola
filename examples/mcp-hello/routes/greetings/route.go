package greetings

import (
	"net/http"

	"github.com/polagonow/pola/routes"
	"mcp-hello/repositories"
	"mcp-hello/services"
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

// POST /greetings
func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	var entity repositories.Greeting
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

// PUT /greetings
func (r *Route) PUT(w http.ResponseWriter, req *http.Request) {
	id, err := routes.ParseUintID(req)
	if err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var entity repositories.Greeting
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

// PATCH /greetings
func (r *Route) PATCH(w http.ResponseWriter, req *http.Request) {
	id, err := routes.ParseUintID(req)
	if err != nil {
		routes.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var entity repositories.Greeting
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

// DELETE /greetings
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

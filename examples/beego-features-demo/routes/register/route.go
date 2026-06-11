package register

import (
	"encoding/json"
	"net/http"

	"beego-features-demo/services"

	"github.com/polagonow/pola/flash"
	"github.com/polagonow/pola/i18n"
	"github.com/polagonow/pola/middleware/session"
)

type Route struct {
	svc *services.UserService
}

func NewRoute(svc *services.UserService) *Route {
	return &Route{svc: svc}
}

func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid JSON"})
		return
	}

	user, err := r.svc.Register(req.Context(), input.Username, input.Password, input.DisplayName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	session.Set(req.Context(), "user_id", user.ID)
	session.Set(req.Context(), "username", user.Username)
	flash.Set(req.Context(), "success", i18n.T(req.Context(), "register_success"))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":       user.ID,
		"username": user.Username,
	})
}

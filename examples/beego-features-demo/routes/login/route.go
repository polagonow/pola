package login

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
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid JSON"})
		return
	}

	user, err := r.svc.Authenticate(req.Context(), input.Username, input.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid credentials"})
		return
	}

	session.Set(req.Context(), "user_id", user.ID)
	session.Set(req.Context(), "username", user.Username)
	flash.Set(req.Context(), "success", i18n.T(req.Context(), "login_success"))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       user.ID,
		"username": user.Username,
	})
}

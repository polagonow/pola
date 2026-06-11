package contacts

import (
	"encoding/json"
	"net/http"

	"github.com/polagonow/pola/validation"
)

type Route struct{}

type Contact struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone" validate:"omitempty,e164"`
}

var v = validation.New()

func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	var input Contact
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := v.Validate(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{"errors": err})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"contact": input,
	})
}

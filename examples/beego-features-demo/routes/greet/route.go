package greet

import (
	"encoding/json"
	"net/http"

	"github.com/polagonow/pola/i18n"
)

type Route struct{}

func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
	name := req.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"locale":   i18n.Locale(req.Context()),
		"greeting": i18n.T(req.Context(), "greeting", name),
	})
}

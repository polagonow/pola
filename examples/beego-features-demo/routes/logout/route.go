package logout

import (
	"encoding/json"
	"net/http"

	"github.com/polagonow/pola/flash"
	"github.com/polagonow/pola/i18n"
	"github.com/polagonow/pola/middleware/session"
)

type Route struct{}

func (r *Route) POST(w http.ResponseWriter, req *http.Request) {
	session.Clear(req.Context())
	flash.Set(req.Context(), "success", i18n.T(req.Context(), "logout_success"))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

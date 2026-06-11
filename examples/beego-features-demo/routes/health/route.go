package health

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

type Route struct{}

func (r *Route) GET(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"go":        runtime.Version(),
	})
}

package health

import (
	"net/http"
	"runtime"
	"time"

	"github.com/polagonow/pola/core"
)

type Route struct{}

// GET /health → returns server status as JSON
func (r *Route) GET(c core.Context) error {
	return c.JSON(http.StatusOK, core.M{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"go":        runtime.Version(),
	})
}

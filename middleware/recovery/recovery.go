// Package recovery provides a panic-recovery Middleware.
package recovery

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/polagonow/pola/core"
)

type mw struct{ log core.Logger }

// New creates a panic recovery middleware.
func New(log core.Logger) core.Middleware { return &mw{log: log} }

func (m *mw) Name() string { return "recovery" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				m.log.Error("panic recovered",
					"error", fmt.Sprintf("%v", rec),
					"stack", string(debug.Stack()),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

package flash

import (
	"context"
	"net/http"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/middleware/session"
)

const flashSessionKey = "_flash"

type mw struct{}

func New() core.Middleware {
	return &mw{}
}

func (m *mw) Name() string { return "flash" }

func (m *mw) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := session.Get(r.Context())
		var flashes map[string]string
		if raw, ok := data[flashSessionKey]; ok {
			if fm, ok := raw.(map[string]string); ok {
				flashes = fm
				session.Delete(r.Context(), flashSessionKey)
			}
		}
		if flashes == nil {
			flashes = make(map[string]string)
		}
		ctx := context.WithValue(r.Context(), core.FlashContextKey, flashes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

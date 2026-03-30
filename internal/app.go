package internal

import (
	"net/http"

	"github.com/polagonow/pola/core"
)

func newApp(cfg *core.Config, registry *core.Registry, handler http.Handler) *core.App {
	return core.NewApp(cfg, registry, handler)
}

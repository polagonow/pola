package internal

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
)

func newApp(cfg *core.Config, injector do.Injector, handler http.Handler) *core.App {
	return core.NewApp(cfg, injector, handler)
}

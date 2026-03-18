package internal

import (
	"net/http"

	"github.com/polagonow/pola/core"
)

// newApp constructs a core.App with the given handler wired in.
// core.App.handler is unexported, so we use the exported SetHandler hook
// that core exposes for exactly this purpose.
func newApp(cfg *core.Config, reg *core.Registry, handler http.Handler) *core.App {
	return core.NewApp(cfg, reg, handler)
}

package react

import (
	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
	"github.com/polagonow/pola/shell"
)

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.Renderer, error) {
			return New(), nil
		})
		do.Provide(i, func(_ do.Injector) (core.HTMLShell, error) {
			return shell.New(`<div id="root"></div>`), nil
		})
	})
}

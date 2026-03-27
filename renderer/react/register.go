package react

import (
	samberdo "github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
	"github.com/polagonow/pola/shell"
)

func init() {
	di.Stage(func(i samberdo.Injector) {
		samberdo.Provide(i, func(_ samberdo.Injector) (core.Renderer, error) {
			return New(), nil
		})
		samberdo.Provide(i, func(_ samberdo.Injector) (core.HTMLShell, error) {
			return shell.New(`<div id="root"></div>`), nil
		})
	})
}

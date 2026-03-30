package routes

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
	"github.com/samber/do/v2"
)

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.APIRouter, error) {
			return New(), nil
		})
	})
}

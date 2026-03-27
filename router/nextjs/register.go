//go:build nextjs

package nextjs

import (
	samberdo "github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

func init() {
	di.Stage(func(i samberdo.Injector) {
		samberdo.Provide(i, func(_ samberdo.Injector) (core.Router, error) {
			return New(), nil
		})
	})
}

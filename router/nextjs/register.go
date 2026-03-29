//go:build nextjs

package nextjs

import (
	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.Router, error) {
			return New(), nil
		})
	})
}

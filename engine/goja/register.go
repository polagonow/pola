//go:build goja

package goja

import (
	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

// Registered is set to true when the goja build tag is active.
var Registered = true

func init() {
	di.Stage(func(i do.Injector) {
		do.Provide(i, func(_ do.Injector) (core.JSEngine, error) {
			return &Engine{}, nil
		})
	})
}

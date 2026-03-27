//go:build goja

package goja

import (
	samberdo "github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
)

// Registered is set to true when the goja build tag is active.
var Registered = true

func init() {
	di.Stage(func(i samberdo.Injector) {
		samberdo.Provide(i, func(_ samberdo.Injector) (core.JSEngine, error) {
			return &Engine{}, nil
		})
	})
}

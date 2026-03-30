package logging

import (
	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
)

// Plugin returns the request-logging middleware plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "logging",
		Fn: func(r *core.Registry) {
			log, err := do.Invoke[core.Logger](r.Injector())
			if err != nil {
				return
			}
			r.AddMiddleware(New(log))
		},
	}
}

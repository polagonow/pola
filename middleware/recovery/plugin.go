package recovery

import (
	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
)

// Plugin returns the panic-recovery middleware plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "recovery",
		Fn: func(r *core.Registry) {
			log, err := do.Invoke[core.Logger](r.Injector())
			if err != nil {
				return
			}
			r.AddMiddleware(New(log))
		},
	}
}

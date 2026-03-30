package recovery

import "github.com/polagonow/pola/core"

// Plugin returns the panic-recovery middleware plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "recovery",
		Fn: func(r *core.Registry) {
			log, err := core.Invoke[core.Logger](r)
			if err != nil {
				return
			}
			r.AddMiddleware(New(log))
		},
	}
}

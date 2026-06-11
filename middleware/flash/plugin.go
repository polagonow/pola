package flash

import "github.com/polagonow/pola/core"

func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "flash",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New())
		},
	}
}

package locale

import "github.com/polagonow/pola/core"

func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "locale",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

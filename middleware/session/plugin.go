package session

import "github.com/polagonow/pola/core"

func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "session",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

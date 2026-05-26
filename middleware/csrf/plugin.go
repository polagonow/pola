package csrf

import "github.com/polagonow/pola/core"

// Plugin returns the CSRF protection middleware plugin.
func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "csrf",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

package securityheaders

import "github.com/polagonow/pola/core"

// Plugin returns the security headers middleware plugin.
func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "securityheaders",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

package routes

import "github.com/polagonow/pola/core"

// Plugin returns the Go API routes plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "routes",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.APIRouter](r, New())
		},
	}
}

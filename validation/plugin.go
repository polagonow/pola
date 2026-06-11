package validation

import "github.com/polagonow/pola/core"

func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "validation",
		Fn: func(r *core.Registry) {
			// Validation is available as a utility; no middleware registration needed
		},
	}
}

package sass

import "github.com/polagonow/pola/core"

// Plugin returns the Sass CSS processor plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "sass",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.CSS](r, New())
		},
	}
}

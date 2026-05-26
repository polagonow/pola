package tailwind

import "github.com/polagonow/pola/core"

// Plugin returns the Tailwind CSS processor plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "tailwind",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.CSS](r, New())
		},
	}
}

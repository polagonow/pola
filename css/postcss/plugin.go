package postcss

import "github.com/polagonow/pola/core"

// Plugin returns the PostCSS processor plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "postcss",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.CSS](r, New())
		},
	}
}

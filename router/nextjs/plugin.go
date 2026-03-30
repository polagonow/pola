package nextjs

import "github.com/polagonow/pola/core"

// Plugin returns the Next.js-style file router plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "nextjs",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Router](r, New())
		},
	}
}

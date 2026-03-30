package moderncquickjs

import "github.com/polagonow/pola/core"

// Plugin returns the Modern C QuickJS engine plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "moderncquickjs",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.JSEngine](r, NewEngine())
		},
	}
}

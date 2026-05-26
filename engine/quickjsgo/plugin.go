package quickjsgo

import "github.com/polagonow/pola/core"

// Plugin returns the QuickJS (CGO) JS engine plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "quickjsgo",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.JSEngine](r, NewEngine())
		},
	}
}

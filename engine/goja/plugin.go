package goja

import "github.com/polagonow/pola/core"

// Plugin returns the Goja JS engine plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "goja",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.JSEngine](r, &Engine{})
		},
	}
}

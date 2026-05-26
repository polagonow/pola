package memory

import "github.com/polagonow/pola/core"

// Plugin returns the in-memory cache plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "memory-cache",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Cache](r, MustNew(defaultSize))
		},
	}
}

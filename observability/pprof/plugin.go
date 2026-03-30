package pprof

import "github.com/polagonow/pola/core"

// Plugin returns the pprof profiling plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "pprof",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Pprof](r, New())
		},
	}
}

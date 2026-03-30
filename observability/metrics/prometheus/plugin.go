package prometheus

import "github.com/polagonow/pola/core"

// Plugin returns the Prometheus metrics plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "prometheus",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Metrics](r, New())
		},
	}
}

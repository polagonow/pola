package std

import "github.com/polagonow/pola/core"

// Plugin registers the std (net/http) web framework as a factory in the DI
// container. It is the default; the pipeline also falls back to std when no
// WebFrameworkFactory is registered.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "webframework:std",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.WebFrameworkFactory](r, func() core.WebFramework { return New() })
		},
	}
}

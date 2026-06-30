package gin

import "github.com/polagonow/pola/core"

// Plugin registers the Gin web framework as a factory in the DI container.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "webframework:gin",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.WebFrameworkFactory](r, func() core.WebFramework { return New() })
		},
	}
}

package chi

import "github.com/polagonow/pola/core"

// Plugin registers the chi web framework as a factory in the DI container.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "webframework:chi",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.WebFrameworkFactory](r, func() core.WebFramework { return New() })
		},
	}
}

package echo

import "github.com/polagonow/pola/core"

// Plugin registers the Echo web framework as a factory in the DI container.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "webframework:echo",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.WebFrameworkFactory](r, func() core.WebFramework { return New() })
		},
	}
}

package storage

import "github.com/polagonow/pola/core"

// Plugin returns a storage plugin that registers a pre-constructed Storage in the DI container.
func Plugin(s Storage) core.Plugin {
	return core.PluginFunc{
		PluginName: "storage",
		Fn: func(r *core.Registry) {
			core.ProvideValue[Storage](r, s)
		},
	}
}

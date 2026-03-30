package esbuild

import "github.com/polagonow/pola/core"

// Plugin returns the esbuild bundler plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "esbuild",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.Bundler](r, New())
		},
	}
}

package reactesbuild

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/react"
)

// Plugin returns a composite plugin that registers the React renderer and
// its esbuild-specific BundlePluginProvider. Use this instead of react.Plugin()
// when the esbuild bundler is in use.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "react-esbuild",
		Fn: func(r *core.Registry) {
			// Register the React renderer and HTML shell.
			react.Plugin().Register(r)
			// Register the esbuild-specific plugin provider.
			core.ProvideValue[core.BundlePluginProvider](r, &pluginProvider{})
		},
	}
}

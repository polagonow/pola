// Package nativerscesbuild provides esbuild-specific build plugins for the
// nativersc renderer, keeping the renderer itself free of esbuild dependencies.
package nativerscesbuild

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/renderer/nativersc"
)

// Plugin returns a composite plugin that registers the nativersc renderer and
// its esbuild-specific BundlePluginProvider. Use this when the esbuild bundler is
// in use (selected via Polafile renderer = "nativersc").
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "nativersc-esbuild",
		Fn: func(r *core.Registry) {
			nativersc.Plugin().Register(r)
			core.ProvideValue[core.BundlePluginProvider](r, &pluginProvider{})
		},
	}
}

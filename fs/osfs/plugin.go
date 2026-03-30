package osfs

import "github.com/polagonow/pola/core"

// Plugin returns the OS filesystem plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "osfs",
		Fn: func(r *core.Registry) {
			core.ProvideValue[core.FS](r, New("."))
			core.ProvideValue[core.AssetServerFactory](r, func(publicDir string) core.AssetServer {
				return &osAssetServer{dir: publicDir}
			})
		},
	}
}

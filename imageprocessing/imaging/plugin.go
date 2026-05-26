package imaging

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/imageprocessing"
)

// Plugin returns the image processing plugin configured with the
// disintegration/imaging adapter. It registers the adapter in the DI
// container and then delegates to the core imageprocessing plugin.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "imageprocessing-imaging",
		Fn: func(r *core.Registry) {
			core.ProvideValue[imageprocessing.ImageProcessor](r, New())
			imageprocessing.Plugin().Register(r)
		},
	}
}

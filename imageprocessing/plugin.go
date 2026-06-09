package imageprocessing

import "github.com/polagonow/pola/core"

// Plugin returns the image processing plugin. It reads configuration from
// environment variables (set by the CLI from Polafile defaults):
//
//	POLA_IMAGE_PROCESSING_PATH       — URL prefix (default "/_pola/image")
//	POLA_IMAGE_PROCESSING_MAX_WIDTH  — max output width (default 4096)
//	POLA_IMAGE_PROCESSING_MAX_HEIGHT — max output height (default 4096)
//	POLA_IMAGE_PROCESSING_FORMAT     — default output format (default "jpeg")
//
// The ImageProcessor adapter must be registered in the DI container before
// this plugin runs (e.g. by the imaging adapter plugin).
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "imageprocessing",
		Fn: func(r *core.Registry) {
			processor, err := core.Invoke[ImageProcessor](r)
			if err != nil {
				return
			}
			svc := New(processor, DefaultConfig())
			r.AddMiddleware(svc)
			r.AddInjector(svc)
		},
	}
}

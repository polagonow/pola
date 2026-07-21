package cors

import "github.com/polagonow/pola/core"

// Plugin returns the CORS middleware plugin. Register it directly:
//
//	pola.Use(cors.Plugin(
//	    cors.WithAllowedOrigins("https://app.example.com"),
//	))
func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "cors",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

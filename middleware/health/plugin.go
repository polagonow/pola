package health

import "github.com/polagonow/pola/core"

// Plugin returns the health middleware plugin. Register it directly:
//
//	pola.Use(health.Plugin(
//	    health.WithCheck("db", func(ctx context.Context) error { return db.PingContext(ctx) }),
//	))
func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "health",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

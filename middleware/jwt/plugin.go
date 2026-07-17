package jwt

import "github.com/polagonow/pola/core"

// Plugin returns the JWT-cookie session middleware plugin. Enable it
// declaratively with a jwt {} block in Polafile.hcl, or register it directly:
//
//	pola.Use(jwt.Plugin(jwt.WithSecret([]byte(os.Getenv("AUTH_SECRET")))))
func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "jwt",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

package requireauth

import "github.com/polagonow/pola/core"

// Plugin returns the route-protection middleware plugin. Enable it declaratively
// with a protect {} block in Polafile.hcl, or register it directly:
//
//	pola.Use(requireauth.Plugin(
//	    requireauth.WithSecret([]byte(os.Getenv("AUTH_SECRET"))),
//	    requireauth.WithProtect("/dashboard"),
//	    requireauth.WithRedirect("/sign-in"),
//	))
func Plugin(opts ...Option) core.Plugin {
	return core.PluginFunc{
		PluginName: "require-auth",
		Fn: func(r *core.Registry) {
			r.AddMiddleware(New(opts...))
		},
	}
}

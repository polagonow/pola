package redis

import (
	"net"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/env"
)

// Plugin returns the Redis cache plugin. Connection settings are read from the
// POLA_CACHE_* environment variables that the Polafile `cache { ... }` block
// populates (host, port, password, db), defaulting to localhost:6379, db 0.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "redis-cache",
		Fn: func(r *core.Registry) {
			host := env.String("POLA_CACHE_HOST", "localhost")
			port := env.String("POLA_CACHE_PORT", "6379")
			core.ProvideValue[core.Cache](r, MustNew(
				WithAddr(net.JoinHostPort(host, port)),
				WithPassword(env.String("POLA_CACHE_PASSWORD", "")),
				WithDB(env.Int("POLA_CACHE_DB", 0)),
			))
		},
	}
}

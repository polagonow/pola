// Package env provides POLA_* environment variable parsing for the Pola framework.
package env

import envparser "github.com/caarlos0/env/v6"

// Env holds all POLA_* environment variables.
type Env struct {
	VM          string `env:"POLA_VM"           envDefault:"goja"`
	Bundler     string `env:"POLA_BUNDLER"      envDefault:"esbuild"`
	Renderer    string `env:"POLA_RENDERER"     envDefault:"react"`
	Router      string `env:"POLA_ROUTER"       envDefault:"nextjs"`
	CSS         string `env:"POLA_CSS"          envDefault:"none"`
	WebAppPath  string `env:"POLA_WEBAPP_PATH"  envDefault:"./web"`
	Env         string `env:"POLA_ENV"           envDefault:"production"`
	EmbedAssets bool   `env:"POLA_EMBED"        envDefault:"true"`

	// Metrics/pprof default to the reserved /_pola/ namespace (see core/reserved).
	// Struct tags cannot reference constants, so the literals are kept in sync
	// with reserved.Metrics / reserved.Pprof.
	MetricsEnabled bool   `env:"POLA_METRICS"       envDefault:"false"`
	MetricsPath    string `env:"POLA_METRICS_PATH"  envDefault:"/_pola/metrics"`

	PprofEnabled bool   `env:"POLA_PPROF"      envDefault:"false"`
	PprofPath    string `env:"POLA_PPROF_PATH" envDefault:"/_pola/pprof"`

	TracingEnabled bool `env:"POLA_TRACING" envDefault:"false"`

	PublicDir string `env:"POLA_PUBLIC_DIR"   envDefault:""`
	BuildOnly bool   `env:"POLA_BUILD_ONLY"  envDefault:"false"`
	// SeedOnly makes pola.Ready() run registered database seeders and then exit,
	// without starting the HTTP server. Set by `pola db seed`.
	SeedOnly bool `env:"POLA_SEED_ONLY" envDefault:"false"`

	Port string `env:"PORT" envDefault:"3000"`

	Address string `env:"POLA_ADDRESS" envDefault:""`
}

// IsDev reports whether the environment is set to development mode.
func (e *Env) IsDev() bool { return e.Env == "development" }

// Load parses POLA_* environment variables into an Env struct.
func Load() (*Env, error) {
	e := new(Env)
	if err := envparser.Parse(e); err != nil {
		return nil, err
	}
	return e, nil
}

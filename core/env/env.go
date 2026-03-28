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
	WebAppPath  string `env:"POLA_WEBAPP_PATH"  envDefault:"./app"`
	Dev         bool   `env:"POLA_DEV"          envDefault:"false"`
	EmbedAssets bool   `env:"POLA_EMBED"        envDefault:"true"`

	MetricsEnabled bool   `env:"POLA_METRICS"       envDefault:"false"`
	MetricsPath    string `env:"POLA_METRICS_PATH"  envDefault:"/metrics"`

	PprofEnabled bool   `env:"POLA_PPROF"      envDefault:"false"`
	PprofPath    string `env:"POLA_PPROF_PATH" envDefault:"/debug/pprof"`

	TracingEnabled bool `env:"POLA_TRACING" envDefault:"false"`

	PublicDir  string `env:"POLA_PUBLIC_DIR"   envDefault:""`
	BuildOnly bool   `env:"POLA_BUILD_ONLY"  envDefault:"false"`

	Port string `env:"PORT" envDefault:"3000"`

	Address string `env:"POLA_ADDRESS" envDefault:""`
}

// Load parses POLA_* environment variables into an Env struct.
func Load() (*Env, error) {
	e := new(Env)
	if err := envparser.Parse(e); err != nil {
		return nil, err
	}
	return e, nil
}

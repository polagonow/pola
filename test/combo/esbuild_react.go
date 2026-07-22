// Package combo registers bundler+renderer+engine combinations for the e2e test suite.
// Each file registers one combination via fixture.Register.
//
// To add a new combination: create a new file here, implement AppFixture,
// and call fixture.Register from init().
// Tests automatically run every registered fixture.
package combo

import (
	"context"
	"sync"
	"testing"

	gojalib "github.com/dop251/goja"

	"github.com/polagonow/pola"
	"github.com/polagonow/pola/bundler/esbuild"
	"github.com/polagonow/pola/cache/memory"
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/css/tailwind"
	"github.com/polagonow/pola/engine/goja"
	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/fs/osfs"
	"github.com/polagonow/pola/logger/slog"
	"github.com/polagonow/pola/middleware/logging"
	"github.com/polagonow/pola/middleware/recovery"
	"github.com/polagonow/pola/observability/metrics/prometheus"
	"github.com/polagonow/pola/observability/tracing/otel"
	reactesbuild "github.com/polagonow/pola/renderer/react/esbuild"
	"github.com/polagonow/pola/router/nextjs"
	"github.com/polagonow/pola/test/fixture"
)

var testPlugins = []core.Plugin{
	goja.Plugin(),
	esbuild.Plugin(),
	reactesbuild.Plugin(),
	tailwind.Plugin(),
	nextjs.Plugin(),
	osfs.Plugin(),
	slog.Plugin(),
	logging.Plugin(),
	recovery.Plugin(),
	memory.Plugin(),
	prometheus.Plugin(),
	otel.Plugin(),
	// Register test injector for DI-backed service calls in e2e tests.
	core.PluginFunc{
		PluginName: "test-fixture",
		Fn: func(r *core.Registry) {
			r.AddInjector(fixture.SharedInjector())
		},
	},
}

func init() {
	fixture.Register(&esbuildReactGojaFixture{})
}

type esbuildReactGojaFixture struct {
	once sync.Once
	app  *core.App
	err  error
}

func (f *esbuildReactGojaFixture) Name() string         { return "goja:react:esbuild" }
func (f *esbuildReactGojaFixture) EngineName() string   { return "goja" }
func (f *esbuildReactGojaFixture) RendererName() string { return "react" }
func (f *esbuildReactGojaFixture) BundlerName() string  { return "esbuild" }

func (f *esbuildReactGojaFixture) GetApp(t *testing.T) *core.App {
	t.Helper()
	f.once.Do(func() {
		builder := pola.NewApp(core.WithWebAppDir(fixture.AppDir))
		builder.Use(testPlugins...)
		f.app, f.err = pola.BuildApp(context.Background(), builder)
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

func (f *esbuildReactGojaFixture) NewPolyfill(_ *testing.T) fixture.PolyfillFixture {
	return &gojaPolyfillFixture{rt: gojalib.New()}
}

type gojaPolyfillFixture struct{ rt *gojalib.Runtime }

func (p *gojaPolyfillFixture) Enable() error {
	reg := polyfill.DefaultRegistry()
	for _, src := range reg.Get(
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.WebpackRequire,
	) {
		if _, err := p.rt.RunString(src.Source); err != nil {
			return err
		}
	}
	return nil
}

func (p *gojaPolyfillFixture) Eval(src string) error {
	_, err := p.rt.RunString(src)
	return err
}

// Package combo registers bundler+renderer+engine combinations for the e2e test suite.
// Each file registers one combination via fixture.Register.
//
// To add a new combination: create a new file here, implement AppFixture,
// and call fixture.Register from init().
// Tests automatically run every registered fixture.
//
//go:build goja && esbuild && react && nextjs
package combo

import (
	"sync"
	"testing"

	gojalib "github.com/dop251/goja"

	pola "github.com/polagonow/pola"
	"github.com/polagonow/pola/core"
	gojaengine "github.com/polagonow/pola/engine/goja"
	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"

	// Import to trigger init() registration of each plugin.
	_ "github.com/polagonow/pola/bundler/esbuild"
	_ "github.com/polagonow/pola/fs/osfs"
	_ "github.com/polagonow/pola/logger/slog"
	_ "github.com/polagonow/pola/renderer/react"
	_ "github.com/polagonow/pola/router/nextjs"
)

func init() { fixture.Register(&esbuildReactGojaFixture{}) }

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
		f.app, f.err = pola.New(&core.Config{
			WebAppPath: fixture.AppDir,
			Registry: &core.Registry{
				Engine:    gojaengine.NewEngine(),
				Injectors: []core.RuntimeInjector{fixture.SharedInjector()},
			},
		})
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

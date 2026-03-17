package vm

import (
	"sync"
	"testing"

	sobeklib "github.com/grafana/sobek"

	esbuild "gojsx/bundler/esbuild"
	"gojsx/framework"
	"gojsx/framework/contract"
	react "gojsx/render/react"
	"gojsx/test/fixture"
	sobekvm "gojsx/vm/sobek"
	sobekpolyfill "gojsx/vm/sobek/polyfill"
)

func init() {
	fixture.Register(&sobekFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type sobekFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *sobekFixture) Name() string     { return "sobek:react:esbuild" }
func (f *sobekFixture) VMName() string   { return "sobek" }
func (f *sobekFixture) Renderer() string { return "react" }
func (f *sobekFixture) Bundler() string  { return "esbuild" }

func (f *sobekFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:          fixture.AppDir,
			GlobalBridge:    fixture.SharedBridge(),
			NewVM:           func(b []byte) (framework.VMFactory, error) { return sobekvm.NewVMFactory(b) },
			Bundler:         esbuild.NewBundler(),
			RendererFactory: func(pool framework.VMPool, protocol framework.StreamProtocol, bridge contract.BridgeConfig) framework.Renderer {
				return react.NewVMRenderer(pool, protocol, bridge)
			},
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

func (f *sobekFixture) NewPolyfill(_ *testing.T) fixture.PolyfillFixture {
	return &sobekPolyfillFixture{rt: sobeklib.New()}
}

type sobekPolyfillFixture struct{ rt *sobeklib.Runtime }

func (f *sobekPolyfillFixture) Enable() error { return sobekpolyfill.Enable(f.rt) }
func (f *sobekPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

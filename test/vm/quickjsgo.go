package vm

import (
	"sync"
	"testing"

	quickjs "github.com/buke/quickjs-go"

	esbuild "gojsx/bundler/esbuild"
	"gojsx/framework"
	"gojsx/framework/contract"
	react "gojsx/render/react"
	"gojsx/test/fixture"
	quickjsgovm "gojsx/vm/quickjsgo"
	quickjsgopolyfill "gojsx/vm/quickjsgo/polyfill"
)

func init() {
	fixture.Register(&quickjsgoFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type quickjsgoFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *quickjsgoFixture) Name() string     { return "quickjsgo:react:esbuild" }
func (f *quickjsgoFixture) VMName() string   { return "quickjsgo" }
func (f *quickjsgoFixture) Renderer() string { return "react" }
func (f *quickjsgoFixture) Bundler() string  { return "esbuild" }

func (f *quickjsgoFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:          fixture.AppDir,
			GlobalBridge:    fixture.SharedBridge(),
			NewVM:           func(b []byte) (framework.VMFactory, error) { return quickjsgovm.NewQuickJSVMFactory(b) },
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

func (f *quickjsgoFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	t.Cleanup(func() {
		ctx.Close()
		rt.Close()
	})
	return &quickjsgoPolyfillFixture{rt: rt, ctx: ctx}
}

type quickjsgoPolyfillFixture struct {
	rt  *quickjs.Runtime
	ctx *quickjs.Context
}

func (f *quickjsgoPolyfillFixture) Enable() error { return quickjsgopolyfill.Enable(f.ctx) }
func (f *quickjsgoPolyfillFixture) Eval(src string) error {
	ret := f.ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		return f.ctx.Exception()
	}
	return nil
}

package fixtures

import (
	"sync"
	"testing"

	quickjs "github.com/buke/quickjs-go"

	"gojsx/framework"
	e2efixture "gojsx/test/e2e/fixture"
	polyfilltest "gojsx/test/polyfill"
	quickjsgovm "gojsx/vm/quickjsgo"
	quickjsgopolyfill "gojsx/vm/quickjsgo/polyfill"
)

func init() {
	e2efixture.Register(&quickjsgoAppFixture{})
	polyfilltest.Register(&newQuickjsgoPolyfillFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type quickjsgoAppFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *quickjsgoAppFixture) Name()     string { return "quickjsgo:react:esbuild" }
func (f *quickjsgoAppFixture) VMName()   string { return "quickjsgo" }
func (f *quickjsgoAppFixture) Renderer() string { return "react" }
func (f *quickjsgoAppFixture) Bundler()  string { return "esbuild" }

func (f *quickjsgoAppFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       e2efixture.AppDir,
			GlobalBridge: e2efixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return quickjsgovm.NewQuickJSVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

type newQuickjsgoPolyfillFixture struct {
	rt  *quickjs.Runtime
	ctx *quickjs.Context
}

func (f *newQuickjsgoPolyfillFixture) Name() string { return "quickjsgo" }

func (f *newQuickjsgoPolyfillFixture) New(t *testing.T) polyfilltest.Fixture {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	t.Cleanup(func() {
		ctx.Close()
		rt.Close()
	})
	return &newQuickjsgoPolyfillFixture{rt: rt, ctx: ctx}
}

func (f *newQuickjsgoPolyfillFixture) Enable() error { return quickjsgopolyfill.Enable(f.ctx) }
func (f *newQuickjsgoPolyfillFixture) Eval(src string) error {
	ret := f.ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		return f.ctx.Exception()
	}
	return nil
}

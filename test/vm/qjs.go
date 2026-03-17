package vm

import (
	"sync"
	"testing"

	qjslib "github.com/fastschema/qjs"

	"gojsx/framework"
	"gojsx/test/fixture"
	qjsvm "gojsx/vm/qjs"
	qjspolyfill "gojsx/vm/qjs/polyfill"
)

func init() {
	fixture.Register(&qjsFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type qjsFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *qjsFixture) Name() string     { return "qjs:react:esbuild" }
func (f *qjsFixture) VMName() string   { return "qjs" }
func (f *qjsFixture) Renderer() string { return "react" }
func (f *qjsFixture) Bundler() string  { return "esbuild" }

func (f *qjsFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       fixture.AppDir,
			GlobalBridge: fixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return qjsvm.NewFastSchemaQJSVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

func (f *qjsFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	rt, err := qjslib.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Close)
	return &qjsPolyfillFixture{ctx: rt.Context()}
}

type qjsPolyfillFixture struct{ ctx *qjslib.Context }

func (f *qjsPolyfillFixture) Enable() error { return qjspolyfill.Enable(f.ctx) }
func (f *qjsPolyfillFixture) Eval(src string) error {
	val, err := f.ctx.Eval("test.js", qjslib.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

package fixtures

import (
	"sync"
	"testing"

	qjslib "github.com/fastschema/qjs"

	"gojsx/framework"
	e2efixture "gojsx/test/e2e/fixture"
	polyfilltest "gojsx/test/polyfill"
	qjsvm "gojsx/vm/qjs"
	qjspolyfill "gojsx/vm/qjs/polyfill"
)

func init() {
	e2efixture.Register(&qjsAppFixture{})
	polyfilltest.Register(&newQjsPolyfillFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type qjsAppFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *qjsAppFixture) Name()     string { return "qjs:react:esbuild" }
func (f *qjsAppFixture) VMName()   string { return "qjs" }
func (f *qjsAppFixture) Renderer() string { return "react" }
func (f *qjsAppFixture) Bundler()  string { return "esbuild" }

func (f *qjsAppFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       e2efixture.AppDir,
			GlobalBridge: e2efixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return qjsvm.NewFastSchemaQJSVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

type newQjsPolyfillFixture struct {
	ctx *qjslib.Context
}

func (f *newQjsPolyfillFixture) Name() string { return "qjs" }

func (f *newQjsPolyfillFixture) New(t *testing.T) polyfilltest.Fixture {
	rt, err := qjslib.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Close)
	return &newQjsPolyfillFixture{ctx: rt.Context()}
}

func (f *newQjsPolyfillFixture) Enable() error { return qjspolyfill.Enable(f.ctx) }
func (f *newQjsPolyfillFixture) Eval(src string) error {
	val, err := f.ctx.Eval("test.js", qjslib.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

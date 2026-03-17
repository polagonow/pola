package vm

import (
	"sync"
	"testing"

	mquickjs "modernc.org/quickjs"

	"gojsx/framework"
	"gojsx/test/fixture"
	moderncvm "gojsx/vm/moderncquickjs"
	moderncpolyfill "gojsx/vm/moderncquickjs/polyfill"
)

func init() {
	fixture.Register(&moderncquickjsFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type moderncquickjsFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *moderncquickjsFixture) Name() string     { return "moderncquickjs:react:esbuild" }
func (f *moderncquickjsFixture) VMName() string   { return "moderncquickjs" }
func (f *moderncquickjsFixture) Renderer() string { return "react" }
func (f *moderncquickjsFixture) Bundler() string  { return "esbuild" }

func (f *moderncquickjsFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       fixture.AppDir,
			GlobalBridge: fixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return moderncvm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

func (f *moderncquickjsFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	vm, err := mquickjs.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vm.Close() })
	return &moderncquickjsPolyfillFixture{vm: vm}
}

type moderncquickjsPolyfillFixture struct{ vm *mquickjs.VM }

func (f *moderncquickjsPolyfillFixture) Enable() error { return moderncpolyfill.Enable(f.vm) }
func (f *moderncquickjsPolyfillFixture) Eval(src string) error {
	_, err := f.vm.Eval(src, mquickjs.EvalGlobal)
	return err
}

package fixtures

import (
	"sync"
	"testing"

	mquickjs "modernc.org/quickjs"

	"gojsx/framework"
	e2efixture "gojsx/test/e2e/fixture"
	polyfilltest "gojsx/test/polyfill"
	moderncvm "gojsx/vm/moderncquickjs"
	moderncpolyfill "gojsx/vm/moderncquickjs/polyfill"
)

func init() {
	e2efixture.Register(&moderncquickjsAppFixture{})
	polyfilltest.Register(&newModerncquickjsPolyfillFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type moderncquickjsAppFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *moderncquickjsAppFixture) Name()     string { return "moderncquickjs:react:esbuild" }
func (f *moderncquickjsAppFixture) VMName()   string { return "moderncquickjs" }
func (f *moderncquickjsAppFixture) Renderer() string { return "react" }
func (f *moderncquickjsAppFixture) Bundler()  string { return "esbuild" }

func (f *moderncquickjsAppFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       e2efixture.AppDir,
			GlobalBridge: e2efixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return moderncvm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

type newModerncquickjsPolyfillFixture struct{ vm *mquickjs.VM }

func (f *newModerncquickjsPolyfillFixture) Name() string { return "moderncquickjs" }

func (f *newModerncquickjsPolyfillFixture) New(t *testing.T) polyfilltest.Fixture {
	vm, err := mquickjs.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vm.Close() })
	return &newModerncquickjsPolyfillFixture{vm: vm}
}

func (f *newModerncquickjsPolyfillFixture) Enable() error { return moderncpolyfill.Enable(f.vm) }
func (f *newModerncquickjsPolyfillFixture) Eval(src string) error {
	_, err := f.vm.Eval(src, mquickjs.EvalGlobal)
	return err
}

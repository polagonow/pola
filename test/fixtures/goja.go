package fixtures

import (
	"sync"
	"testing"

	gojalib "github.com/dop251/goja"

	"gojsx/framework"
	e2efixture "gojsx/test/e2e/fixture"
	polyfilltest "gojsx/test/polyfill"
	gojavm "gojsx/vm/goja"
	gojapolyfill "gojsx/vm/goja/polyfill"
)

func init() {
	e2efixture.Register(&gojaAppFixture{})
	polyfilltest.Register(&newGojaPolyfillFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type gojaAppFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *gojaAppFixture) Name()     string { return "goja:react:esbuild" }
func (f *gojaAppFixture) VMName()   string { return "goja" }
func (f *gojaAppFixture) Renderer() string { return "react" }
func (f *gojaAppFixture) Bundler()  string { return "esbuild" }

func (f *gojaAppFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       e2efixture.AppDir,
			GlobalBridge: e2efixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return gojavm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

type newGojaPolyfillFixture struct{ rt *gojalib.Runtime }

func (f *newGojaPolyfillFixture) Name() string { return "goja" }

func (f *newGojaPolyfillFixture) New(_ *testing.T) polyfilltest.Fixture {
	return &newGojaPolyfillFixture{rt: gojalib.New()}
}

func (f *newGojaPolyfillFixture) Enable() error { return gojapolyfill.Enable(f.rt) }
func (f *newGojaPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

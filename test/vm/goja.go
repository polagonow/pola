package vm

import (
	"sync"
	"testing"

	gojalib "github.com/dop251/goja"

	"gojsx/framework"
	"gojsx/test/fixture"
	gojavm "gojsx/vm/goja"
	gojapolyfill "gojsx/vm/goja/polyfill"
)

func init() {
	fixture.Register(&gojaFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type gojaFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *gojaFixture) Name() string     { return "goja:react:esbuild" }
func (f *gojaFixture) VMName() string   { return "goja" }
func (f *gojaFixture) Renderer() string { return "react" }
func (f *gojaFixture) Bundler() string  { return "esbuild" }

func (f *gojaFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       fixture.AppDir,
			GlobalBridge: fixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return gojavm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

func (f *gojaFixture) NewPolyfill(_ *testing.T) fixture.PolyfillFixture {
	return &gojaPolyfillFixture{rt: gojalib.New()}
}

type gojaPolyfillFixture struct{ rt *gojalib.Runtime }

func (f *gojaPolyfillFixture) Enable() error { return gojapolyfill.Enable(f.rt) }
func (f *gojaPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

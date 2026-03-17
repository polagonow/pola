package fixtures

import (
	"sync"
	"testing"

	v8 "rogchap.com/v8go"

	"gojsx/framework"
	e2efixture "gojsx/test/e2e/fixture"
	polyfilltest "gojsx/test/polyfill"
	v8govm "gojsx/vm/v8go"
	v8polyfill "gojsx/vm/v8go/polyfill"
)

func init() {
	e2efixture.Register(&v8goAppFixture{})
	polyfilltest.Register(&newV8goPolyfillFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type v8goAppFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *v8goAppFixture) Name()     string { return "v8go:react:esbuild" }
func (f *v8goAppFixture) VMName()   string { return "v8go" }
func (f *v8goAppFixture) Renderer() string { return "react" }
func (f *v8goAppFixture) Bundler()  string { return "esbuild" }

func (f *v8goAppFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       e2efixture.AppDir,
			GlobalBridge: e2efixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return v8govm.NewV8VMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

type newV8goPolyfillFixture struct {
	iso *v8.Isolate
	ctx *v8.Context
}

func (f *newV8goPolyfillFixture) Name() string { return "v8go" }

func (f *newV8goPolyfillFixture) New(t *testing.T) polyfilltest.Fixture {
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return &newV8goPolyfillFixture{iso: iso, ctx: v8.NewContext(iso)}
}

func (f *newV8goPolyfillFixture) Enable() error { return v8polyfill.Enable(f.ctx) }
func (f *newV8goPolyfillFixture) Eval(src string) error {
	_, err := f.ctx.RunScript(src, "test.js")
	return err
}

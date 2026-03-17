package vm

import (
	"sync"
	"testing"

	v8 "rogchap.com/v8go"

	"gojsx/framework"
	"gojsx/test/fixture"
	v8govm "gojsx/vm/v8go"
	v8polyfill "gojsx/vm/v8go/polyfill"
)

func init() {
	fixture.Register(&v8goFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type v8goFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *v8goFixture) Name() string     { return "v8go:react:esbuild" }
func (f *v8goFixture) VMName() string   { return "v8go" }
func (f *v8goFixture) Renderer() string { return "react" }
func (f *v8goFixture) Bundler() string  { return "esbuild" }

func (f *v8goFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       fixture.AppDir,
			GlobalBridge: fixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return v8govm.NewV8VMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

func (f *v8goFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return &v8goPolyfillFixture{iso: iso, ctx: v8.NewContext(iso)}
}

type v8goPolyfillFixture struct {
	iso *v8.Isolate
	ctx *v8.Context
}

func (f *v8goPolyfillFixture) Enable() error { return v8polyfill.Enable(f.ctx) }
func (f *v8goPolyfillFixture) Eval(src string) error {
	_, err := f.ctx.RunScript(src, "test.js")
	return err
}

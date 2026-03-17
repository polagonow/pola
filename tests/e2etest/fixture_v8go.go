package e2etest

import (
	"sync"
	"testing"

	"gojsx/framework"
	v8govm "gojsx/vm/v8go"
)

func init() { register(&v8goFixture{}) }

type v8goFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *v8goFixture) Name()     string { return "v8go:react:esbuild" }
func (f *v8goFixture) VMName()   string { return "v8go" }
func (f *v8goFixture) Renderer() string { return "react" }
func (f *v8goFixture) Bundler()  string { return "esbuild" }

func (f *v8goFixture) getApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       "../../ui/apps/blog",
			GlobalBridge: SharedBridge(),
			NewVM: func(b []byte) (framework.VMFactory, error) { return v8govm.NewV8VMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

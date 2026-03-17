package e2etest

import (
	"sync"
	"testing"

	"gojsx/framework"
	moderncvm "gojsx/vm/moderncquickjs"
)

func init() { register(&moderncquickjsFixture{}) }

type moderncquickjsFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *moderncquickjsFixture) Name()     string { return "moderncquickjs:react:esbuild" }
func (f *moderncquickjsFixture) VMName()   string { return "moderncquickjs" }
func (f *moderncquickjsFixture) Renderer() string { return "react" }
func (f *moderncquickjsFixture) Bundler()  string { return "esbuild" }

func (f *moderncquickjsFixture) getApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       "../../ui/apps/blog",
			GlobalBridge: SharedBridge(),
			NewVM: func(b []byte) (framework.VMFactory, error) { return moderncvm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

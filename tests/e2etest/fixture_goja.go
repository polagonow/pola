package e2etest

import (
	"sync"
	"testing"

	_ "gojsx/bundler/esbuild"
	"gojsx/framework"
	_ "gojsx/framework/assets/disk"
	_ "gojsx/render/react"
	_ "gojsx/render/react/discovery/nextjs"
	_ "gojsx/render/react/shell"
	gojavm "gojsx/vm/goja"
)

func init() { register(&gojaFixture{}) }

type gojaFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *gojaFixture) Name()     string { return "goja:react:esbuild" }
func (f *gojaFixture) VMName()   string { return "goja" }
func (f *gojaFixture) Renderer() string { return "react" }
func (f *gojaFixture) Bundler()  string { return "esbuild" }

func (f *gojaFixture) getApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       "../../ui/apps/blog",
			GlobalBridge: SharedBridge(),
			NewVM: func(b []byte) (framework.VMFactory, error) { return gojavm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

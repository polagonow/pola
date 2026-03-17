package e2etest

import (
	"sync"
	"testing"

	"gojsx/framework"
	quickjsgovm "gojsx/vm/quickjsgo"
)

func init() { register(&quickjsgoFixture{}) }

type quickjsgoFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *quickjsgoFixture) Name()     string { return "quickjsgo:react:esbuild" }
func (f *quickjsgoFixture) VMName()   string { return "quickjsgo" }
func (f *quickjsgoFixture) Renderer() string { return "react" }
func (f *quickjsgoFixture) Bundler()  string { return "esbuild" }

func (f *quickjsgoFixture) getApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       "../../ui/apps/blog",
			GlobalBridge: SharedBridge(),
			NewVM: func(b []byte) (framework.VMFactory, error) { return quickjsgovm.NewQuickJSVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

package e2etest

import (
	"sync"
	"testing"

	"gojsx/framework"
	sobekvm "gojsx/vm/sobek"
)

func init() { register(&sobekFixture{}) }

type sobekFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *sobekFixture) Name()     string { return "sobek:react:esbuild" }
func (f *sobekFixture) VMName()   string { return "sobek" }
func (f *sobekFixture) Renderer() string { return "react" }
func (f *sobekFixture) Bundler()  string { return "esbuild" }

func (f *sobekFixture) getApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       "../../ui/apps/blog",
			GlobalBridge: SharedBridge(),
			NewVM: func(b []byte) (framework.VMFactory, error) { return sobekvm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

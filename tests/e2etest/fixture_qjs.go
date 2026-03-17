package e2etest

import (
	"sync"
	"testing"

	"gojsx/framework"
	qjsvm "gojsx/vm/qjs"
)

func init() { register(&qjsFixture{}) }

type qjsFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *qjsFixture) Name()     string { return "qjs:react:esbuild" }
func (f *qjsFixture) VMName()   string { return "qjs" }
func (f *qjsFixture) Renderer() string { return "react" }
func (f *qjsFixture) Bundler()  string { return "esbuild" }

func (f *qjsFixture) getApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       "../../ui/apps/blog",
			GlobalBridge: SharedBridge(),
			NewVM: func(b []byte) (framework.VMFactory, error) { return qjsvm.NewFastSchemaQJSVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

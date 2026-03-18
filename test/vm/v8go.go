package vm

import (
	"testing"

	v8 "rogchap.com/v8go"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/test/fixture"
	v8govm "github.com/polagonow/pola/vm/v8go"
	v8polyfill "github.com/polagonow/pola/vm/v8go/polyfill"
)

func init() { fixture.RegisterVM(&v8goVMFixture{}) }

type v8goVMFixture struct{}

func (f *v8goVMFixture) VMName() string { return "v8go" }
func (f *v8goVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
	return func(b []byte) (framework.VMFactory, error) { return v8govm.NewV8VMFactory(b) }
}
func (f *v8goVMFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
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

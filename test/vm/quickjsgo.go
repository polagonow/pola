package vm

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"

	"gojsx/framework"
	"gojsx/test/fixture"
	quickjsgovm "gojsx/vm/quickjsgo"
	quickjsgopolyfill "gojsx/vm/quickjsgo/polyfill"
)

func init() { fixture.RegisterVM(&quickjsgoVMFixture{}) }

type quickjsgoVMFixture struct{}

func (f *quickjsgoVMFixture) VMName() string { return "quickjsgo" }
func (f *quickjsgoVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
	return func(b []byte) (framework.VMFactory, error) { return quickjsgovm.NewQuickJSVMFactory(b) }
}
func (f *quickjsgoVMFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	t.Cleanup(func() {
		ctx.Close()
		rt.Close()
	})
	return &quickjsgoPolyfillFixture{rt: rt, ctx: ctx}
}

type quickjsgoPolyfillFixture struct {
	rt  *quickjs.Runtime
	ctx *quickjs.Context
}

func (f *quickjsgoPolyfillFixture) Enable() error { return quickjsgopolyfill.Enable(f.ctx) }
func (f *quickjsgoPolyfillFixture) Eval(src string) error {
	ret := f.ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		return f.ctx.Exception()
	}
	return nil
}

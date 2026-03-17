package polyfilltest

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"

	quickjsgopolyfill "gojsx/vm/quickjsgo/polyfill"
)

func init() {
	register("quickjsgo", newQuickjsgoFixture)
}

type quickjsgoFixture struct {
	rt  *quickjs.Runtime
	ctx *quickjs.Context
}

func newQuickjsgoFixture(t *testing.T) Fixture {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	t.Cleanup(func() {
		ctx.Close()
		rt.Close()
	})
	return &quickjsgoFixture{rt: rt, ctx: ctx}
}

func (f *quickjsgoFixture) Name() string { return "quickjsgo" }

func (f *quickjsgoFixture) Enable() error {
	return quickjsgopolyfill.Enable(f.ctx)
}

func (f *quickjsgoFixture) Eval(src string) error {
	ret := f.ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		return f.ctx.Exception()
	}
	return nil
}

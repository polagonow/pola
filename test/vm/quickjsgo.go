//go:build quickjsgo

package vm

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"

	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"
)

func init() {
	fixture.RegisterPolyfillVM("quickjsgo", func(t *testing.T) fixture.PolyfillFixture {
		rt := quickjs.NewRuntime()
		ctx := rt.NewContext()
		t.Cleanup(func() {
			ctx.Close()
			rt.Close()
		})
		return &quickjsgoPolyfillFixture{rt: rt, ctx: ctx}
	})
}

type quickjsgoPolyfillFixture struct {
	rt  *quickjs.Runtime
	ctx *quickjs.Context
}

func (f *quickjsgoPolyfillFixture) Enable() error {
	reg := polyfill.DefaultRegistry()
	for _, src := range reg.Get(
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.Promise,
		polyfill.WebpackRequire,
	) {
		ret := f.ctx.Eval(src.Source, quickjs.EvalFileName("polyfill.js"))
		defer ret.Free()
		if ret.IsException() {
			return f.ctx.Exception()
		}
	}
	return nil
}

func (f *quickjsgoPolyfillFixture) Eval(src string) error {
	ret := f.ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		return f.ctx.Exception()
	}
	return nil
}

//go:build qjs

package vm

import (
	"testing"

	qjslib "github.com/fastschema/qjs"

	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"
)

func init() {
	fixture.RegisterPolyfillVM("qjs", func(t *testing.T) fixture.PolyfillFixture {
		rt, err := qjslib.New()
		if err != nil {
			return &errPolyfillFixture{err: err}
		}
		t.Cleanup(rt.Close)
		return &qjsPolyfillFixture{ctx: rt.Context()}
	})
}

type qjsPolyfillFixture struct{ ctx *qjslib.Context }

func (f *qjsPolyfillFixture) Enable() error {
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
		val, err := f.ctx.Eval("polyfill.js", qjslib.Code(src.Source))
		if val != nil {
			val.Free()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *qjsPolyfillFixture) Eval(src string) error {
	val, err := f.ctx.Eval("test.js", qjslib.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

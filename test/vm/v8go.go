//go:build v8go

package vm

import (
	"testing"

	v8 "rogchap.com/v8go"

	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"
)

func init() {
	fixture.RegisterPolyfillVM("v8go", func(t *testing.T) fixture.PolyfillFixture {
		iso := v8.NewIsolate()
		t.Cleanup(iso.Dispose)
		return &v8goPolyfillFixture{iso: iso, ctx: v8.NewContext(iso)}
	})
}

type v8goPolyfillFixture struct {
	iso *v8.Isolate
	ctx *v8.Context
}

func (f *v8goPolyfillFixture) Enable() error {
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
		if _, err := f.ctx.RunScript(src.Source, "polyfill.js"); err != nil {
			return err
		}
	}
	return nil
}

func (f *v8goPolyfillFixture) Eval(src string) error {
	_, err := f.ctx.RunScript(src, "test.js")
	return err
}

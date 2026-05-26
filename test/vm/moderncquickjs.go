//go:build moderncquickjs

package vm

import (
	"testing"

	mquickjs "modernc.org/quickjs"

	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"
)

func init() {
	fixture.RegisterPolyfillVM("moderncquickjs", func(t *testing.T) fixture.PolyfillFixture {
		vm, err := mquickjs.NewVM()
		if err != nil {
			return &errPolyfillFixture{err: err}
		}
		t.Cleanup(func() { _ = vm.Close() })
		return &moderncquickjsPolyfillFixture{vm: vm}
	})
}

type moderncquickjsPolyfillFixture struct{ vm *mquickjs.VM }

func (f *moderncquickjsPolyfillFixture) Enable() error {
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
		if _, err := f.vm.Eval(src.Source, mquickjs.EvalGlobal); err != nil {
			return err
		}
	}
	return nil
}

func (f *moderncquickjsPolyfillFixture) Eval(src string) error {
	_, err := f.vm.Eval(src, mquickjs.EvalGlobal)
	return err
}

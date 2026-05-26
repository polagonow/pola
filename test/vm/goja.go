package vm

import (
	"testing"

	gojalib "github.com/dop251/goja"

	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"
)

func init() {
	fixture.RegisterPolyfillVM("goja", func(_ *testing.T) fixture.PolyfillFixture {
		return &gojaPolyfillFixture{rt: gojalib.New()}
	})
}

type gojaPolyfillFixture struct{ rt *gojalib.Runtime }

func (f *gojaPolyfillFixture) Enable() error {
	reg := polyfill.DefaultRegistry()
	for _, src := range reg.Get(
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.MessageChannel,
		polyfill.ReadableStream,
		polyfill.AbortController,
		polyfill.WebpackRequire,
	) {
		if _, err := f.rt.RunString(src.Source); err != nil {
			return err
		}
	}
	return nil
}

func (f *gojaPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

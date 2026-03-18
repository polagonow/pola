//go:build sobek

package vm

import (
	"testing"

	sobeklib "github.com/grafana/sobek"

	"github.com/polagonow/pola/engine/polyfill"
	"github.com/polagonow/pola/test/fixture"
)

func init() {
	fixture.RegisterPolyfillVM("sobek", func(_ *testing.T) fixture.PolyfillFixture {
		return &sobekPolyfillFixture{rt: sobeklib.New()}
	})
}

type sobekPolyfillFixture struct{ rt *sobeklib.Runtime }

func (f *sobekPolyfillFixture) Enable() error {
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

func (f *sobekPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

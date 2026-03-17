package polyfilltest

import (
	"testing"

	sobeklib "github.com/grafana/sobek"

	sobekpolyfill "gojsx/vm/sobek/polyfill"
)

func init() {
	register("sobek", newSobekFixture)
}

type sobekFixture struct {
	rt *sobeklib.Runtime
}

func newSobekFixture(t *testing.T) Fixture {
	return &sobekFixture{rt: sobeklib.New()}
}

func (f *sobekFixture) Name() string { return "sobek" }

func (f *sobekFixture) Enable() error {
	return sobekpolyfill.Enable(f.rt)
}

func (f *sobekFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

package polyfilltest

import (
	"testing"

	gojalib "github.com/dop251/goja"

	gojapolyfill "gojsx/vm/goja/polyfill"
)

func init() {
	register("goja", newGojaFixture)
}

type gojaFixture struct {
	rt *gojalib.Runtime
}

func newGojaFixture(t *testing.T) Fixture {
	return &gojaFixture{rt: gojalib.New()}
}

func (f *gojaFixture) Name() string { return "goja" }

func (f *gojaFixture) Enable() error {
	return gojapolyfill.Enable(f.rt)
}

func (f *gojaFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

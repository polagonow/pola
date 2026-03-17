package vm

import (
	"testing"

	gojalib "github.com/dop251/goja"

	"gojsx/framework"
	"gojsx/test/fixture"
	gojavm "gojsx/vm/goja"
	gojapolyfill "gojsx/vm/goja/polyfill"
)

func init() { fixture.RegisterVM(&gojaVMFixture{}) }

type gojaVMFixture struct{}

func (f *gojaVMFixture) VMName() string { return "goja" }
func (f *gojaVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
	return func(b []byte) (framework.VMFactory, error) { return gojavm.NewVMFactory(b) }
}
func (f *gojaVMFixture) NewPolyfill(_ *testing.T) fixture.PolyfillFixture {
	return &gojaPolyfillFixture{rt: gojalib.New()}
}

type gojaPolyfillFixture struct{ rt *gojalib.Runtime }

func (f *gojaPolyfillFixture) Enable() error { return gojapolyfill.Enable(f.rt) }
func (f *gojaPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

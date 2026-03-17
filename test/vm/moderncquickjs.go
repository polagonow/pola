package vm

import (
	"testing"

	mquickjs "modernc.org/quickjs"

	"gojsx/framework"
	"gojsx/test/fixture"
	moderncvm "gojsx/vm/moderncquickjs"
	moderncpolyfill "gojsx/vm/moderncquickjs/polyfill"
)

func init() { fixture.RegisterVM(&moderncquickjsVMFixture{}) }

type moderncquickjsVMFixture struct{}

func (f *moderncquickjsVMFixture) VMName() string { return "moderncquickjs" }
func (f *moderncquickjsVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
	return func(b []byte) (framework.VMFactory, error) { return moderncvm.NewVMFactory(b) }
}
func (f *moderncquickjsVMFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	vm, err := mquickjs.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vm.Close() })
	return &moderncquickjsPolyfillFixture{vm: vm}
}

type moderncquickjsPolyfillFixture struct{ vm *mquickjs.VM }

func (f *moderncquickjsPolyfillFixture) Enable() error { return moderncpolyfill.Enable(f.vm) }
func (f *moderncquickjsPolyfillFixture) Eval(src string) error {
	_, err := f.vm.Eval(src, mquickjs.EvalGlobal)
	return err
}

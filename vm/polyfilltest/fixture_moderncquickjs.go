package polyfilltest

import (
	"testing"

	mquickjs "modernc.org/quickjs"

	moderncpolyfill "gojsx/vm/moderncquickjs/polyfill"
)

func init() {
	register("moderncquickjs", newModerncquickjsFixture)
}

type moderncquickjsFixture struct {
	vm *mquickjs.VM
}

func newModerncquickjsFixture(t *testing.T) Fixture {
	vm, err := mquickjs.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vm.Close() })
	return &moderncquickjsFixture{vm: vm}
}

func (f *moderncquickjsFixture) Name() string { return "moderncquickjs" }

func (f *moderncquickjsFixture) Enable() error {
	return moderncpolyfill.Enable(f.vm)
}

func (f *moderncquickjsFixture) Eval(src string) error {
	_, err := f.vm.Eval(src, mquickjs.EvalGlobal)
	return err
}

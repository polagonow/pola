package polyfilltest

import (
	"testing"

	v8 "rogchap.com/v8go"

	v8polyfill "gojsx/vm/v8go/polyfill"
)

func init() {
	register("v8go", newV8goFixture)
}

type v8goFixture struct {
	iso *v8.Isolate
	ctx *v8.Context
}

func newV8goFixture(t *testing.T) Fixture {
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return &v8goFixture{iso: iso, ctx: v8.NewContext(iso)}
}

func (f *v8goFixture) Name() string { return "v8go" }

func (f *v8goFixture) Enable() error {
	return v8polyfill.Enable(f.ctx)
}

func (f *v8goFixture) Eval(src string) error {
	_, err := f.ctx.RunScript(src, "test.js")
	return err
}

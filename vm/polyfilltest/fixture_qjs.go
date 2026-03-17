package polyfilltest

import (
	"testing"

	qjs "github.com/fastschema/qjs"

	qjspolyfill "gojsx/vm/qjs/polyfill"
)

func init() {
	register("qjs", newQjsFixture)
}

type qjsFixture struct {
	ctx *qjs.Context
}

func newQjsFixture(t *testing.T) Fixture {
	rt, err := qjs.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Close)
	return &qjsFixture{ctx: rt.Context()}
}

func (f *qjsFixture) Name() string { return "qjs" }

func (f *qjsFixture) Enable() error {
	return qjspolyfill.Enable(f.ctx)
}

func (f *qjsFixture) Eval(src string) error {
	val, err := f.ctx.Eval("test.js", qjs.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

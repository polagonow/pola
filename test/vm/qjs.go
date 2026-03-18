package vm

import (
	"testing"

	qjslib "github.com/fastschema/qjs"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/test/fixture"
	qjsvm "github.com/polagonow/pola/vm/qjs"
	qjspolyfill "github.com/polagonow/pola/vm/qjs/polyfill"
)

func init() { fixture.RegisterVM(&qjsVMFixture{}) }

type qjsVMFixture struct{}

func (f *qjsVMFixture) VMName() string { return "qjs" }
func (f *qjsVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
	return func(b []byte) (framework.VMFactory, error) { return qjsvm.NewFastSchemaQJSVMFactory(b) }
}
func (f *qjsVMFixture) NewPolyfill(t *testing.T) fixture.PolyfillFixture {
	rt, err := qjslib.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Close)
	return &qjsPolyfillFixture{ctx: rt.Context()}
}

type qjsPolyfillFixture struct{ ctx *qjslib.Context }

func (f *qjsPolyfillFixture) Enable() error { return qjspolyfill.Enable(f.ctx) }
func (f *qjsPolyfillFixture) Eval(src string) error {
	val, err := f.ctx.Eval("test.js", qjslib.Code(src))
	if val != nil {
		val.Free()
	}
	return err
}

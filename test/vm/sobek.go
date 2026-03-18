package vm

import (
	"testing"

	sobeklib "github.com/grafana/sobek"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/test/fixture"
	sobekvm "github.com/polagonow/pola/vm/sobek"
	sobekpolyfill "github.com/polagonow/pola/vm/sobek/polyfill"
)

func init() { fixture.RegisterVM(&sobekVMFixture{}) }

type sobekVMFixture struct{}

func (f *sobekVMFixture) VMName() string { return "sobek" }
func (f *sobekVMFixture) VMFactory() func([]byte) (framework.VMFactory, error) {
	return func(b []byte) (framework.VMFactory, error) { return sobekvm.NewVMFactory(b) }
}
func (f *sobekVMFixture) NewPolyfill(_ *testing.T) fixture.PolyfillFixture {
	return &sobekPolyfillFixture{rt: sobeklib.New()}
}

type sobekPolyfillFixture struct{ rt *sobeklib.Runtime }

func (f *sobekPolyfillFixture) Enable() error { return sobekpolyfill.Enable(f.rt) }
func (f *sobekPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

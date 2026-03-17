package fixtures

import (
	"sync"
	"testing"

	sobeklib "github.com/grafana/sobek"

	"gojsx/framework"
	e2efixture "gojsx/test/e2e/fixture"
	polyfilltest "gojsx/test/polyfill"
	sobekvm "gojsx/vm/sobek"
	sobekpolyfill "gojsx/vm/sobek/polyfill"
)

func init() {
	e2efixture.Register(&sobekAppFixture{})
	polyfilltest.Register(&newSobekPolyfillFixture{})
}

// ── e2e ──────────────────────────────────────────────────────────────────────

type sobekAppFixture struct {
	once sync.Once
	app  *framework.App
	err  error
}

func (f *sobekAppFixture) Name()     string { return "sobek:react:esbuild" }
func (f *sobekAppFixture) VMName()   string { return "sobek" }
func (f *sobekAppFixture) Renderer() string { return "react" }
func (f *sobekAppFixture) Bundler()  string { return "esbuild" }

func (f *sobekAppFixture) GetApp(t *testing.T) *framework.App {
	t.Helper()
	f.once.Do(func() {
		f.app, f.err = (&framework.Config{
			AppDir:       e2efixture.AppDir,
			GlobalBridge: e2efixture.SharedBridge(),
			NewVM:        func(b []byte) (framework.VMFactory, error) { return sobekvm.NewVMFactory(b) },
		}).Build()
	})
	if f.err != nil {
		t.Fatalf("%s: build failed: %v", f.Name(), f.err)
	}
	return f.app
}

// ── polyfill ─────────────────────────────────────────────────────────────────

type newSobekPolyfillFixture struct{ rt *sobeklib.Runtime }

func (f *newSobekPolyfillFixture) Name() string { return "sobek" }

func (f *newSobekPolyfillFixture) New(_ *testing.T) polyfilltest.Fixture {
	return &newSobekPolyfillFixture{rt: sobeklib.New()}
}

func (f *newSobekPolyfillFixture) Enable() error { return sobekpolyfill.Enable(f.rt) }
func (f *newSobekPolyfillFixture) Eval(src string) error {
	_, err := f.rt.RunString(src)
	return err
}

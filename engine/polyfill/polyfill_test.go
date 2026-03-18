package polyfill_test

import (
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/engine/polyfill"
)

func TestDefaultRegistry_NotNil(t *testing.T) {
	reg := polyfill.DefaultRegistry()
	if reg == nil {
		t.Fatal("DefaultRegistry returned nil")
	}
}

func TestDefaultRegistry_AllIDsHaveSource(t *testing.T) {
	reg := polyfill.DefaultRegistry()

	ids := []core.PolyfillID{
		polyfill.Promise,
		polyfill.ReadableStream,
		polyfill.MessageChannel,
		polyfill.AbortController,
		polyfill.MicrotaskQueue,
		polyfill.TextEncoding,
		polyfill.WebpackRequire,
	}

	sources := reg.Get(ids...)
	if len(sources) != len(ids) {
		t.Fatalf("expected %d polyfill sources, got %d", len(ids), len(sources))
	}

	for _, src := range sources {
		if src.Source == "" {
			t.Errorf("polyfill %q has empty source", src.ID)
		}
	}
}

func TestRegister_PopulatesExternalRegistry(t *testing.T) {
	reg := core.NewPolyfillRegistry()
	polyfill.Register(reg)

	sources := reg.Get(polyfill.MicrotaskQueue, polyfill.WebpackRequire)
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources after Register, got %d", len(sources))
	}
}

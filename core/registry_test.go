package core_test

import (
	"testing"

	"github.com/polagonow/pola/core"
)

func TestPolyfillRegistry(t *testing.T) {
	reg := core.NewPolyfillRegistry()
	reg.Register(core.PolyfillSource{ID: "promise", Source: "// promise polyfill"})

	results := reg.Get("promise")
	if len(results) != 1 {
		t.Fatalf("expected 1 polyfill, got %d", len(results))
	}
	if results[0].Source != "// promise polyfill" {
		t.Fatalf("unexpected source: %s", results[0].Source)
	}
}

func TestPolyfillRegistryMissing(t *testing.T) {
	reg := core.NewPolyfillRegistry()
	results := reg.Get("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 polyfills for unknown ID, got %d", len(results))
	}
}

func TestRegistryFillDefaultsNoopFallback(t *testing.T) {
	reg := &core.Registry{}
	if err := reg.FillDefaults(); err != nil {
		t.Fatalf("FillDefaults failed: %v", err)
	}
	// Logger should be noop, not nil
	if reg.Logger == nil {
		t.Fatal("Logger should not be nil after FillDefaults")
	}
	// Metrics should be noop
	if reg.Metrics == nil {
		t.Fatal("Metrics should not be nil after FillDefaults")
	}
}

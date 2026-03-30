package core_test

import (
	"testing"

	"github.com/samber/do/v2"

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

func TestRegistryProvideAndInvoke(t *testing.T) {
	r := core.NewRegistry()
	core.ProvideValue[string](r, "hello")

	val, err := do.Invoke[string](r.Injector())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected %q, got %q", "hello", val)
	}
}

func TestRegistryMiddlewareAndInjectors(t *testing.T) {
	r := core.NewRegistry()

	if len(r.Middleware()) != 0 {
		t.Fatal("expected no middleware initially")
	}
	if len(r.RuntimeInjectors()) != 0 {
		t.Fatal("expected no injectors initially")
	}
}

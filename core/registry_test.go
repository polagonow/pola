package core_test

import (
	"testing"

	"github.com/samber/do/v2"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/core/di"
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

func TestDIBuildMissingServicesError(t *testing.T) {
	injector := di.Build()

	// Unregistered services should return errors, not noops.
	if _, err := do.Invoke[core.Logger](injector); err == nil {
		t.Fatal("expected error for unregistered Logger")
	}
	if _, err := do.Invoke[core.Metrics](injector); err == nil {
		t.Fatal("expected error for unregistered Metrics")
	}
	if _, err := do.Invoke[core.Tracer](injector); err == nil {
		t.Fatal("expected error for unregistered Tracer")
	}

	// Framework-internal singletons should always be available.
	if _, err := do.Invoke[*di.MiddlewareCollector](injector); err != nil {
		t.Fatalf("MiddlewareCollector should be registered: %v", err)
	}
	if _, err := do.Invoke[*di.InjectorCollector](injector); err != nil {
		t.Fatalf("InjectorCollector should be registered: %v", err)
	}
	if _, err := do.Invoke[*di.EventBus](injector); err != nil {
		t.Fatalf("EventBus should be registered: %v", err)
	}
}

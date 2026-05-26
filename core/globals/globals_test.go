package globals_test

import (
	"testing"

	"github.com/polagonow/pola/core/globals"
)

func TestGlobalsNotEmpty(t *testing.T) {
	if globals.RequestContext == "" {
		t.Fatal("RequestContext must not be empty")
	}
	if globals.BridgeObject != "__DEPENDENCY_INJECTION__" {
		t.Fatalf("BridgeObject = %q, want __DEPENDENCY_INJECTION__", globals.BridgeObject)
	}
	if globals.RenderFn == "" {
		t.Fatal("RenderFn must not be empty")
	}
}

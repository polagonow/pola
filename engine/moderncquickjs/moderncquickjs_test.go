//go:build moderncquickjs

package moderncquickjs_test

import (
	"context"
	"testing"

	"github.com/polagonow/pola/engine/moderncquickjs"
)

func TestEngine_Name(t *testing.T) {
	eng := moderncquickjs.NewEngine()
	if eng.Name() != "moderncquickjs" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "moderncquickjs")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng := moderncquickjs.NewEngine()
	if len(eng.RequiredPolyfills()) == 0 {
		t.Error("RequiredPolyfills returned empty slice")
	}
}

func TestEngine_NewRuntime_errors(t *testing.T) {
	eng := moderncquickjs.NewEngine()
	_, err := eng.NewRuntime(context.Background())
	if err == nil {
		t.Error("NewRuntime: expected error (use NewSSRPool), got nil")
	}
}

func TestEngine_Registered(t *testing.T) {
	if !moderncquickjs.Registered {
		t.Error("Registered should be true when moderncquickjs build tag is active")
	}
}

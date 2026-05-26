//go:build quickjsgo

package quickjsgo_test

import (
	"context"
	"testing"

	"github.com/polagonow/pola/engine/quickjsgo"
)

const simpleBundle = `var greet = function(name) { return 'hello ' + name; };`

func TestEngine_Name(t *testing.T) {
	eng := quickjsgo.NewEngine()
	if eng.Name() != "quickjsgo" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "quickjsgo")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng := quickjsgo.NewEngine()
	if len(eng.RequiredPolyfills()) == 0 {
		t.Error("RequiredPolyfills returned empty slice")
	}
}

func TestEngine_NewRuntime_errors(t *testing.T) {
	eng := quickjsgo.NewEngine()
	_, err := eng.NewRuntime(context.Background())
	if err == nil {
		t.Error("NewRuntime: expected error (use NewSSRPool), got nil")
	}
}

func TestEngine_Registered(t *testing.T) {
	if !quickjsgo.Registered {
		t.Error("Registered should be true when quickjsgo build tag is active")
	}
}

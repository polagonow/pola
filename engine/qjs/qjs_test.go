//go:build qjs

package qjs_test

import (
	"context"
	"testing"

	engineqjs "github.com/polagonow/pola/engine/qjs"
)

const simpleBundle = `var greet = function(name) { return 'hello ' + name; };`

func TestEngine_Name(t *testing.T) {
	eng := engineqjs.New(simpleBundle)
	if eng.Name() != "qjs" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "qjs")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng := engineqjs.New(simpleBundle)
	if len(eng.RequiredPolyfills()) == 0 {
		t.Error("RequiredPolyfills returned empty slice")
	}
}

func TestRuntime_Eval(t *testing.T) {
	eng := engineqjs.New(simpleBundle)
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()

	result, err := rt.Eval("1 + 2")
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	// qjs returns float64 for numbers
	switch v := result.(type) {
	case float64:
		if v != 3 {
			t.Errorf("Eval = %v, want 3", v)
		}
	case int64:
		if v != 3 {
			t.Errorf("Eval = %v, want 3", v)
		}
	default:
		t.Errorf("unexpected type %T value %v", result, result)
	}
}

func TestRuntime_Set(t *testing.T) {
	eng := engineqjs.New(simpleBundle)
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()

	if err := rt.Set("__testVal__", "pola"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	result, err := rt.Eval("__testVal__")
	if err != nil {
		t.Fatalf("Eval after Set: %v", err)
	}
	if result != "pola" {
		t.Errorf("after Set got %v, want 'pola'", result)
	}
}

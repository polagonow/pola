//go:build v8go

package v8go_test

import (
	"context"
	"testing"

	enginev8 "github.com/polagonow/pola/engine/v8go"
)

const simpleBundle = `var greet = function(name) { return 'hello ' + name; };`

func TestEngine_Name(t *testing.T) {
	eng := enginev8.New(simpleBundle)
	if eng.Name() != "v8go" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "v8go")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng := enginev8.New(simpleBundle)
	if len(eng.RequiredPolyfills()) == 0 {
		t.Error("RequiredPolyfills returned empty slice")
	}
}

func TestRuntime_Eval(t *testing.T) {
	eng := enginev8.New(simpleBundle)
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()

	result, err := rt.Eval("1 + 2")
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != int64(3) {
		t.Errorf("Eval('1+2') = %v, want 3", result)
	}
}

func TestRuntime_Set(t *testing.T) {
	eng := enginev8.New(simpleBundle)
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

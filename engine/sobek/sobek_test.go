//go:build sobek

package sobek_test

import (
	"context"
	"testing"

	enginesob "github.com/polagonow/pola/engine/sobek"
)

const simpleBundle = `var greet = function(name) { return 'hello ' + name; };`

func TestEngine_Name(t *testing.T) {
	eng, err := enginesob.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if eng.Name() != "sobek" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "sobek")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng, err := enginesob.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(eng.RequiredPolyfills()) == 0 {
		t.Error("RequiredPolyfills returned empty slice")
	}
}

func TestRuntime_Eval(t *testing.T) {
	eng, err := enginesob.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
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

func TestRuntime_Call(t *testing.T) {
	eng, err := enginesob.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()

	result, err := rt.Call("greet", "world")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != "hello world" {
		t.Errorf("Call = %v, want 'hello world'", result)
	}
}

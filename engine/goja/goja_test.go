package goja_test

import (
	"context"
	"testing"

	"github.com/polagonow/pola/engine/goja"
)

const simpleBundle = `
var __render__ = function(exportName, propsJSON) { return exportName + ':' + propsJSON; };
var greet = function(name) { return 'hello ' + name; };
`

func TestEngine_CompileBundle(t *testing.T) {
	eng, err := goja.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if eng.Name() != "goja" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "goja")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng, err := goja.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	polyfills := eng.RequiredPolyfills()
	if len(polyfills) == 0 {
		t.Error("RequiredPolyfills returned empty slice")
	}
}

func TestRuntime_Eval(t *testing.T) {
	eng, err := goja.New(simpleBundle)
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
	// Goja exports int64 for integer arithmetic
	if v, ok := result.(int64); !ok || v != 3 {
		t.Errorf("Eval('1+2') = %v (%T), want 3", result, result)
	}
}

func TestRuntime_Call(t *testing.T) {
	eng, err := goja.New(simpleBundle)
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
		t.Errorf("Call('greet', 'world') = %v, want 'hello world'", result)
	}
}

func TestRuntime_Set(t *testing.T) {
	eng, err := goja.New(simpleBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
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

func TestVMPool_AcquireRelease(t *testing.T) {
	pool, err := goja.NewVMPool(simpleBundle, nil)
	if err != nil {
		t.Fatalf("NewVMPool: %v", err)
	}

	rt, err := pool.Acquire()
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if rt == nil {
		t.Fatal("Acquire returned nil")
	}
	pool.Release(rt)
}

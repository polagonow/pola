package node_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/polagonow/pola/engine/node"
)

// skipIfNoNode skips the test if the node binary is not found.
func skipIfNoNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node binary not found in PATH")
	}
}

func TestEngine_Name(t *testing.T) {
	eng := node.New("")
	if eng.Name() != "node" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "node")
	}
}

func TestEngine_RequiredPolyfills(t *testing.T) {
	eng := node.New("")
	if p := eng.RequiredPolyfills(); p != nil {
		t.Errorf("RequiredPolyfills() should be nil for Node, got %v", p)
	}
}

func TestRuntime_Eval(t *testing.T) {
	skipIfNoNode(t)

	eng := node.New("node")
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()

	result, err := rt.Eval("process.stdout.write('hello')")
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != "hello" {
		t.Errorf("Eval output = %q, want %q", result, "hello")
	}
}

func TestRuntime_Set_And_Eval(t *testing.T) {
	skipIfNoNode(t)

	eng := node.New("node")
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()

	if err := rt.Set("__greeting__", "pola"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	result, err := rt.Eval("process.stdout.write(__greeting__)")
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if result != "pola" {
		t.Errorf("Eval after Set = %q, want %q", result, "pola")
	}
}

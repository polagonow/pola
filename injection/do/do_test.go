package do

import (
	"context"
	"strings"
	"testing"

	samberdo "github.com/samber/do/v2"
	"github.com/polagonow/pola/core/globals"
)

// mockRuntime is a test double for core.JSRuntime.
type mockRuntime struct {
	globals map[string]any
}

func (m *mockRuntime) Eval(script string) (any, error)        { return nil, nil }
func (m *mockRuntime) Call(fn string, args ...any) (any, error) { return nil, nil }
func (m *mockRuntime) Set(name string, value any) error {
	m.globals[name] = value
	return nil
}
func (m *mockRuntime) Dispose() {}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{globals: make(map[string]any)}
}

func TestName(t *testing.T) {
	inj := New(samberdo.New())
	if inj.Name() != "do" {
		t.Fatalf("expected 'do', got %q", inj.Name())
	}
}

func TestRegister_And_Inject(t *testing.T) {
	di := samberdo.New()
	inj := New(di)

	called := false
	inj.Register("getUser", func(args []any) (any, error) {
		called = true
		return map[string]any{"id": 1}, nil
	})

	rt := newMockRuntime()
	if err := inj.Inject(context.Background(), rt); err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}

	fns, ok := rt.globals[globals.BridgeObject]
	if !ok {
		t.Fatalf("BridgeObject %q not set on runtime", globals.BridgeObject)
	}

	fnMap, ok := fns.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", fns)
	}

	getUserFn, ok := fnMap["getUser"]
	if !ok {
		t.Fatal("getUser not found in injected map")
	}

	// Call the function to verify it works.
	fn := getUserFn.(func(args []any) (any, error))
	result, err := fn(nil)
	if err != nil {
		t.Fatalf("getUser returned error: %v", err)
	}
	if !called {
		t.Fatal("expected the registered function to be called")
	}
	m, ok := result.(map[string]any)
	if !ok || m["id"] != 1 {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestCapabilities(t *testing.T) {
	inj := New(samberdo.New())
	inj.Register("listProducts", func(args []any) (any, error) { return nil, nil })
	inj.Register("getProduct", func(args []any) (any, error) { return nil, nil })

	caps := inj.Capabilities()
	if len(caps) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(caps))
	}

	names := map[string]bool{}
	for _, c := range caps {
		names[c.Name] = true
	}
	if !names["listProducts"] || !names["getProduct"] {
		t.Fatalf("unexpected capabilities: %v", caps)
	}
}

func TestTypeScriptDeclarations_Empty(t *testing.T) {
	inj := New(samberdo.New())
	decl := inj.TypeScriptDeclarations()
	if !strings.Contains(decl, "No DI functions registered") {
		t.Fatalf("expected no-functions comment, got: %q", decl)
	}
}

func TestTypeScriptDeclarations_WithFunctions(t *testing.T) {
	inj := New(samberdo.New())
	inj.Register("getUser", func(args []any) (any, error) { return nil, nil })

	decl := inj.TypeScriptDeclarations()

	if !strings.Contains(decl, "AUTO-GENERATED") {
		t.Error("missing AUTO-GENERATED header")
	}
	if !strings.Contains(decl, globals.BridgeObject) {
		t.Errorf("missing BridgeObject %q", globals.BridgeObject)
	}
	if !strings.Contains(decl, "getUser") {
		t.Error("missing 'getUser' in declarations")
	}
	if !strings.Contains(decl, "Promise<any>") {
		t.Error("missing Promise<any> in declarations")
	}
}

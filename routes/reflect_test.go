package routes

import (
	"testing"

	"github.com/polagonow/pola/core"
)

// testRoute is a mock route struct with HTTP method handlers.
type testRoute struct {
	getCalled    bool
	postCalled   bool
	putCalled    bool
	deleteCalled bool
}

func (r *testRoute) GET(core.Context) error    { r.getCalled = true; return nil }
func (r *testRoute) POST(core.Context) error   { r.postCalled = true; return nil }
func (r *testRoute) PUT(core.Context) error    { r.putCalled = true; return nil }
func (r *testRoute) DELETE(core.Context) error { r.deleteCalled = true; return nil }

func TestDiscoverActions(t *testing.T) {
	route := &testRoute{}
	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}

	if len(discovered) != 4 {
		t.Fatalf("expected 4 actions, got %d", len(discovered))
	}

	byMethod := make(map[string]discoveredAction)
	for _, d := range discovered {
		byMethod[d.Method] = d
	}

	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		if _, ok := byMethod[method]; !ok {
			t.Errorf("missing action for %s", method)
		}
	}
}

func TestDiscoverActions_InvokesHandler(t *testing.T) {
	route := &testRoute{}
	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}

	h := mount(splitActions("/t", discovered))
	do(h, "GET", "/t")
	if !route.getCalled {
		t.Error("GET handler did not invoke the method")
	}
}

// partialRoute only has some methods.
type partialRoute struct{}

func (r *partialRoute) POST(core.Context) error   { return nil }
func (r *partialRoute) DELETE(core.Context) error { return nil }

func TestDiscoverActions_Partial(t *testing.T) {
	route := &partialRoute{}
	discovered, err := discoverActions(route)
	if err != nil {
		t.Fatalf("discoverActions: %v", err)
	}

	if len(discovered) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(discovered))
	}

	byMethod := make(map[string]bool)
	for _, d := range discovered {
		byMethod[d.Method] = true
	}
	if !byMethod["POST"] || !byMethod["DELETE"] {
		t.Errorf("expected POST and DELETE, got %v", byMethod)
	}
}

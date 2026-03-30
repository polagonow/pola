package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// testRoute is a mock route struct with HTTP method handlers.
type testRoute struct {
	getCalled    bool
	postCalled   bool
	putCalled    bool
	deleteCalled bool
}

func (r *testRoute) GET(w http.ResponseWriter, req *http.Request)    { r.getCalled = true }
func (r *testRoute) POST(w http.ResponseWriter, req *http.Request)   { r.postCalled = true }
func (r *testRoute) PUT(w http.ResponseWriter, req *http.Request)    { r.putCalled = true }
func (r *testRoute) DELETE(w http.ResponseWriter, req *http.Request) { r.deleteCalled = true }

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

	for _, d := range discovered {
		if d.Method == "GET" {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			d.Handler(w, req)
			if !route.getCalled {
				t.Error("GET handler did not invoke the method")
			}
			break
		}
	}
}

// partialRoute only has some methods.
type partialRoute struct{}

func (r *partialRoute) POST(w http.ResponseWriter, req *http.Request)   {}
func (r *partialRoute) DELETE(w http.ResponseWriter, req *http.Request) {}

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

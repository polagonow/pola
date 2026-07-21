package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/routes"
)

func sampleSpecs() routes.RouteSpecs {
	return routes.RouteSpecs{
		{Method: "GET", Pattern: "/users"},
		{Method: "POST", Pattern: "/users"},
		{Method: "GET", Pattern: "/users/:id", Meta: map[string]any{"summary": "Fetch a user", "tags": []string{"users"}}},
		{Method: "GET", Pattern: "/files/:...path"},
	}
}

func TestGenerate(t *testing.T) {
	doc := Generate(sampleSpecs(), Info{Title: "Shop", Version: "1.0.0"})

	if doc.OpenAPI != "3.0.3" || doc.Info.Title != "Shop" {
		t.Errorf("header = %+v", doc)
	}

	// /users has both get and post.
	users := doc.Paths["/users"]
	if _, ok := users["get"]; !ok {
		t.Error("/users missing get")
	}
	if _, ok := users["post"]; !ok {
		t.Error("/users missing post")
	}

	// :id → {id} with a path parameter.
	byID, ok := doc.Paths["/users/{id}"]
	if !ok {
		t.Fatalf("path not converted; got %v", doc.SortedPaths())
	}
	op := byID["get"]
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "id" || op.Parameters[0].In != "path" || !op.Parameters[0].Required {
		t.Errorf("path param = %+v", op.Parameters)
	}
	if op.Summary != "Fetch a user" || len(op.Tags) != 1 || op.Tags[0] != "users" {
		t.Errorf("meta not applied: %+v", op)
	}

	// Catch-all :...path → {path}.
	if _, ok := doc.Paths["/files/{path}"]; !ok {
		t.Errorf("catch-all not converted; got %v", doc.SortedPaths())
	}
}

func TestJSONIsValid(t *testing.T) {
	doc := Generate(sampleSpecs(), Info{Title: "Shop", Version: "1.0.0"})
	b, err := doc.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("generated spec is not valid JSON: %v", err)
	}
	if round["openapi"] != "3.0.3" {
		t.Errorf("openapi field = %v", round["openapi"])
	}
}

func TestServe(t *testing.T) {
	doc := Generate(sampleSpecs(), Info{Title: "Shop", Version: "1.0.0"})
	var reached bool
	h := Serve(doc).Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusTeapot)
	}))

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	spec := get("/openapi.json")
	if spec.Code != http.StatusOK || !strings.Contains(spec.Header().Get("Content-Type"), "json") {
		t.Errorf("spec response = %d %q", spec.Code, spec.Header().Get("Content-Type"))
	}
	if !strings.Contains(spec.Body.String(), `"/users/{id}"`) {
		t.Error("spec body missing a path")
	}

	ui := get("/openapi")
	if ui.Code != http.StatusOK || !strings.Contains(ui.Body.String(), "swagger-ui") {
		t.Errorf("ui response = %d, body swagger? %v", ui.Code, strings.Contains(ui.Body.String(), "swagger-ui"))
	}

	other := get("/api/whatever")
	if !reached || other.Code != http.StatusTeapot {
		t.Errorf("non-openapi path should pass through (reached=%v code=%d)", reached, other.Code)
	}
}

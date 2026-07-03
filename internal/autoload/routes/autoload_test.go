package routes

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const polaPkg = "github.com/polagonow/pola"

func writePkg(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func gen(t *testing.T, dir string) string {
	t.Helper()
	hasStruct, funcs := scanRouteStyle(dir)
	if !hasStruct && len(funcs) == 0 {
		t.Fatalf("no route detected in %s", dir)
	}
	hasCtor, ctorParams, err := discoverRouteCtor(dir, polaPkg)
	if err != nil {
		t.Fatalf("discover ctor: %v", err)
	}
	var depVal = discoverRouteServiceDep(dir, "myapp")
	if hasCtor {
		depVal = nil
	}
	src, err := generateRouteInit("todos", polaPkg, "myapp", depVal, hasStruct, funcs, hasCtor, ctorParams)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "pola_route_init.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated does not parse: %v\n%s", err, src)
	}
	return string(src)
}

func TestRouteRegistryCtor(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, map[string]string{
		"route.go": `package todos
import (
	"net/http"

	"github.com/polagonow/pola/core"
)
type Route struct{}
func NewRoute(r *core.Registry) *Route { return &Route{} }
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {}
`,
	})
	src := gen(t, dir)
	if !strings.Contains(src, "return NewRoute(r)") {
		t.Fatalf("want registry-style call\n%s", src)
	}
}

func TestRouteCtorWithServiceDep(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, map[string]string{
		"route.go": `package todos
import (
	"net/http"

	"github.com/polagonow/pola/core"

	"myapp/services"
)
type Route struct{ svc services.TodoServiceInterface }
func NewRoute(r *core.Registry) *Route { return &Route{svc: core.MustInvoke[services.TodoServiceInterface](r)} }
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {}
`,
	})
	src := gen(t, dir)
	if !strings.Contains(src, "return NewRoute(r)") {
		t.Fatalf("want registry-style call\n%s", src)
	}
}

func TestRouteCtorMixedRejected(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, map[string]string{
		"route.go": `package todos
import (
	"net/http"

	"github.com/polagonow/pola/core"

	"myapp/services"
)
type Route struct{}
func NewRoute(r *core.Registry, svc services.TodoServiceInterface) *Route { return &Route{} }
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {}
`,
	})
	_, _, err := discoverRouteCtor(dir, polaPkg)
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error rejecting mixed signature, got %v", err)
	}
}

func TestRouteCtorZeroParam(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, map[string]string{
		"route.go": `package todos
import "net/http"
type Route struct{}
func NewRoute() *Route { return &Route{} }
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {}
`,
	})
	src := gen(t, dir)
	if !strings.Contains(src, "return NewRoute()") {
		t.Fatalf("want zero-arg call\n%s", src)
	}
}

func TestRouteNoCtorLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, map[string]string{
		"route.go": `package todos
import (
	"net/http"

	"myapp/services"
)
type Route struct{ svc *services.TodoService }
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {}
`,
	})
	src := gen(t, dir)
	if !strings.Contains(src, "svc := core.MustInvoke[*services.TodoService](r)") {
		t.Fatalf("want legacy dep resolution\n%s", src)
	}
	if !strings.Contains(src, "return NewRoute(svc)") {
		t.Fatalf("want NewRoute(svc)\n%s", src)
	}
}

func TestRouteNoCtorNoDeps(t *testing.T) {
	dir := t.TempDir()
	writePkg(t, dir, map[string]string{
		"route.go": `package todos
import "net/http"
type Route struct{}
func (r *Route) GET(w http.ResponseWriter, req *http.Request) {}
`,
	})
	src := gen(t, dir)
	if !strings.Contains(src, "return &Route{}") {
		t.Fatalf("want empty-struct fallback\n%s", src)
	}
}

package ctorscan_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/autoload/ctorscan"
)

const polaPkg = "github.com/polagonow/pola"

func scan(t *testing.T, src, ctorName string) ([]ctorscan.Param, error) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != ctorName {
			continue
		}
		return ctorscan.ScanParams(fd, f, "test.go", polaPkg)
	}
	t.Fatalf("constructor %q not found", ctorName)
	return nil, nil
}

func TestNonRegistryParamRejected_Gorm(t *testing.T) {
	src := `package repos
import "gorm.io/gorm"
func NewUserRepository(db *gorm.DB) *UserRepository { return nil }
`
	_, err := scan(t, src, "NewUserRepository")
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error requiring *core.Registry, got %v", err)
	}
}

func TestNonRegistryParamRejected_Interface(t *testing.T) {
	src := `package services
import "myapp/repositories"
func NewTodoService(repo repositories.TodoRepository) *TodoService { return nil }
`
	_, err := scan(t, src, "NewTodoService")
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error requiring *core.Registry, got %v", err)
	}
}

func TestZeroParamAccepted(t *testing.T) {
	src := `package services
func NewTodoService() *TodoService { return nil }
`
	params, err := scan(t, src, "NewTodoService")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(params) != 0 {
		t.Fatalf("want 0 params, got %d", len(params))
	}
}

func TestRegistryCtor(t *testing.T) {
	src := `package svc
import "github.com/polagonow/pola/core"
func NewFooService(r *core.Registry) *FooService { return nil }
`
	params, err := scan(t, src, "NewFooService")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !params[0].IsRegistry {
		t.Fatalf("want IsRegistry=true, got %+v", params[0])
	}
	if params[0].Import != nil {
		t.Fatalf("want no import (generator adds core header), got %+v", params[0].Import)
	}
}

func TestMixedRegistryAndDepsRejected(t *testing.T) {
	src := `package svc
import (
	"github.com/polagonow/pola/core"
	"myapp/repositories"
)
func NewMixed(r *core.Registry, repo repositories.TodoRepository) *Mixed { return nil }
`
	_, err := scan(t, src, "NewMixed")
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error rejecting mixed signature, got %v", err)
	}
}

func TestAliasedCoreImport(t *testing.T) {
	src := `package svc
import pc "github.com/polagonow/pola/core"
func NewFoo(r *pc.Registry) *Foo { return nil }
`
	params, err := scan(t, src, "NewFoo")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !params[0].IsRegistry {
		t.Fatalf("want IsRegistry=true even with aliased core import")
	}
}

func TestValueRegistryRejected(t *testing.T) {
	src := `package svc
import "github.com/polagonow/pola/core"
func NewFoo(r core.Registry) *Foo { return nil }
`
	_, err := scan(t, src, "NewFoo")
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error mentioning *core.Registry, got %v", err)
	}
}

func TestDotImportRejected(t *testing.T) {
	src := `package svc
import . "myapp/services"
func NewFoo(x FooDep) *Foo { return nil }
`
	_, err := scan(t, src, "NewFoo")
	if err == nil || !strings.Contains(err.Error(), "dot import") {
		t.Fatalf("want dot-import error, got %v", err)
	}
}

func TestUnsupportedExprRejected(t *testing.T) {
	src := `package svc
func NewFoo(cb func() int) *Foo { return nil }
`
	_, err := scan(t, src, "NewFoo")
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("want unsupported error, got %v", err)
	}
}

func TestMultiNameRegistryFieldExpanded(t *testing.T) {
	// Rare, but supported: `func NewFoo(a, b *core.Registry)` — both fields
	// are treated as registry params. Callers get two Params, both IsRegistry.
	src := `package svc
import "github.com/polagonow/pola/core"
func NewFoo(a, b *core.Registry) *Foo { return nil }
`
	params, err := scan(t, src, "NewFoo")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(params) != 2 || !params[0].IsRegistry || !params[1].IsRegistry {
		t.Fatalf("want 2 registry params, got %+v", params)
	}
}

func TestMergeImportsDeduplicates(t *testing.T) {
	imports := []ctorscan.Param{
		{Type: "*gorm.DB", Import: &ctorscan.Import{Path: "gorm.io/gorm"}},
		{Type: "*gorm.DB", Import: &ctorscan.Import{Path: "gorm.io/gorm"}},
		{Type: "repositories.X", Import: &ctorscan.Import{Path: "myapp/repositories"}},
	}
	merged, _ := ctorscan.MergeImports(imports, map[string]struct{}{})
	if len(merged) != 2 {
		t.Fatalf("want 2 merged imports, got %d: %+v", len(merged), merged)
	}
}

func TestMergeImportsSkipsCore(t *testing.T) {
	imports := []ctorscan.Param{
		{Type: "*core.Registry", Import: &ctorscan.Import{Path: polaPkg + "/core"}},
		{Type: "*gorm.DB", Import: &ctorscan.Import{Path: "gorm.io/gorm"}},
	}
	merged, _ := ctorscan.MergeImports(imports, map[string]struct{}{polaPkg + "/core": {}})
	if len(merged) != 1 || merged[0].Path != "gorm.io/gorm" {
		t.Fatalf("want only gorm, got %+v", merged)
	}
}

func TestMergeImportsDisambiguatesAliasCollision(t *testing.T) {
	imports := []ctorscan.Param{
		{Type: "*models.X", Import: &ctorscan.Import{Path: "pkg/a/models"}},
		{Type: "*models.Y", Import: &ctorscan.Import{Path: "pkg/b/models"}},
	}
	merged, renames := ctorscan.MergeImports(imports, map[string]struct{}{})
	if len(merged) != 2 {
		t.Fatalf("want 2 imports, got %d", len(merged))
	}
	if len(renames) != 1 {
		t.Fatalf("want 1 rename, got %+v", renames)
	}
}

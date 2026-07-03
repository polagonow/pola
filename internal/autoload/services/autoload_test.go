package services

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const polaPkg = "github.com/polagonow/pola"

func setupProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module myapp\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func generate(t *testing.T, projectDir string) string {
	t.Helper()
	disco, err := discoverServiceConstructors(projectDir, "services", "myapp", polaPkg)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if disco == nil {
		t.Fatal("discover: nil")
	}
	var buf bytes.Buffer
	if err := svcPluginsTmpl.Execute(&buf, tmplData{PolaPackage: polaPkg, SvcDiscovery: disco}); err != nil {
		t.Fatalf("template: %v", err)
	}
	src := buf.String()
	if _, err := parser.ParseFile(token.NewFileSet(), "pola_svc_plugins.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}
	return src
}

func TestServicesLegacyExplicitDepRejected(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/todo_service.go": `package services
import "myapp/repositories"
type TodoServiceInterface interface{}
type TodoService struct { repo repositories.TodoRepository }
func NewTodoService(repo repositories.TodoRepository) *TodoService { return &TodoService{repo: repo} }
`,
	})
	_, err := discoverServiceConstructors(dir, "services", "myapp", polaPkg)
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error rejecting explicit repo param, got %v", err)
	}
}

func TestServicesRegistryStyleWithInterface(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/todo_service.go": `package services
import (
	"github.com/polagonow/pola/core"
	"myapp/repositories"
)
type TodoServiceInterface interface{}
type TodoService struct{}
func NewTodoService(r *core.Registry) *TodoService {
	return &TodoService{}
}
var _ = repositories.DefaultPerPage
`,
	})
	src := generate(t, dir)
	checks := []string{
		`func TodoServicePlugin() core.Plugin`,
		`return NewTodoService(r), nil`,
		`core.Provide[TodoServiceInterface](r`,
	}
	for _, want := range checks {
		if !strings.Contains(src, want) {
			t.Errorf("want %q\n%s", want, src)
		}
	}
}

func TestServicesRegistryStyle(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/todo_service.go": `package services
import "github.com/polagonow/pola/core"
type TodoServiceInterface interface{}
type TodoService struct{}
func NewTodoService(r *core.Registry) *TodoService { return &TodoService{} }
`,
	})
	src := generate(t, dir)
	if !strings.Contains(src, "return NewTodoService(r), nil") {
		t.Fatalf("want registry-style call\n%s", src)
	}
	if strings.Contains(src, "core.Invoke[") && !strings.Contains(src, "core.Invoke[*TodoService](r)") {
		t.Fatalf("registry-only ctor should not emit dependency Invoke\n%s", src)
	}
}

func TestServicesZeroParam(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/todo_service.go": `package services
type TodoService struct{}
func NewTodoService() *TodoService { return &TodoService{} }
`,
	})
	src := generate(t, dir)
	if !strings.Contains(src, "return NewTodoService(), nil") {
		t.Fatalf("want zero-arg call\n%s", src)
	}
	if strings.Contains(src, "TodoServiceInterface") {
		t.Fatalf("no interface declared, alias must not appear\n%s", src)
	}
}

func TestServicesNoInterface(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/todo_service.go": `package services
import "github.com/polagonow/pola/core"
type TodoService struct{}
func NewTodoService(r *core.Registry) *TodoService { return &TodoService{} }
`,
	})
	src := generate(t, dir)
	if strings.Contains(src, "TodoServiceInterface") {
		t.Fatalf("no interface declared, alias must not appear\n%s", src)
	}
	if !strings.Contains(src, "return NewTodoService(r), nil") {
		t.Fatalf("want service call\n%s", src)
	}
}

func TestServicesNoRepositoriesPackage(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/mailer_service.go": `package services
type MailerService struct{}
func NewMailerService() *MailerService { return &MailerService{} }
`,
	})
	src := generate(t, dir)
	if strings.Contains(src, `"myapp/repositories"`) {
		t.Fatalf("must not import repositories when unused\n%s", src)
	}
}

func TestServicesMixedRegistryAndDepRejected(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"services/todo_service.go": `package services
import (
	"github.com/polagonow/pola/core"
	"myapp/repositories"
)
type TodoService struct{}
func NewTodoService(r *core.Registry, repo repositories.TodoRepository) *TodoService { return &TodoService{} }
`,
	})
	_, err := discoverServiceConstructors(dir, "services", "myapp", polaPkg)
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error rejecting mixed signature, got %v", err)
	}
}

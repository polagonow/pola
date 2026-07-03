package repos

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

// setupProject writes a minimal go.mod plus the given files under a temp dir
// and returns the project directory.
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

func generate(t *testing.T, projectDir, orm string) string {
	t.Helper()
	disco, err := discoverRepositoryRegistrations(projectDir, "repositories", "db/client/ent", orm, "myapp", polaPkg)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if disco == nil {
		t.Fatal("discover: nil discovery")
	}
	var buf bytes.Buffer
	if err := repoPluginsTmpl.Execute(&buf, disco); err != nil {
		t.Fatalf("template: %v", err)
	}
	src := buf.String()
	if _, err := parser.ParseFile(token.NewFileSet(), "pola_repo_plugins.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, src)
	}
	return src
}

func TestReposLegacyExplicitDepRejected(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"repositories/gorm/todo_repository.go": `package gorm
import (
	"gorm.io/gorm"
	"myapp/repositories"
)
type todoRepository struct { db *gorm.DB }
func NewTodoRepository(db *gorm.DB) repositories.TodoRepository { return &todoRepository{db: db} }
`,
	})
	_, err := discoverRepositoryRegistrations(dir, "repositories", "db/client/ent", "gorm", "myapp", polaPkg)
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error rejecting explicit db param, got %v", err)
	}
}

func TestReposRegistryStyle(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"repositories/gorm/todo_repository.go": `package gorm
import (
	"github.com/polagonow/pola/core"
	"myapp/repositories"
)
type todoRepository struct{}
func NewTodoRepository(r *core.Registry) repositories.TodoRepository { return &todoRepository{} }
`,
	})
	src := generate(t, dir, "gorm")
	if !strings.Contains(src, "return NewTodoRepository(r), nil") {
		t.Fatalf("want registry-style call\n%s", src)
	}
	if strings.Contains(src, "core.Invoke[") {
		t.Fatalf("registry-only ctor should not emit any Invoke\n%s", src)
	}
}

func TestReposEntRepositoryOnly(t *testing.T) {
	// The repos autoload emits only repository plugins; the ent client plugin
	// is a separate concern owned by internal/autoload/entclient.
	dir := setupProject(t, map[string]string{
		"repositories/ent/todo_repository.go": `package ent
import (
	"github.com/polagonow/pola/core"
	"myapp/repositories"
)
type todoRepository struct{}
func NewTodoRepository(r *core.Registry) repositories.TodoRepository { return &todoRepository{} }
`,
	})
	src := generate(t, dir, "ent")
	if !strings.Contains(src, `return NewTodoRepository(r), nil`) {
		t.Errorf("want registry-style call\n%s", src)
	}
	if strings.Contains(src, "EntClientPlugin") {
		t.Errorf("repos autoload must not emit EntClientPlugin (owned by entclient autoload)\n%s", src)
	}
}

func TestReposBeegoRegistryStyle(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"repositories/beego/todo_repository.go": `package beego
import (
	"github.com/polagonow/pola/core"
	"myapp/repositories"
)
type todoRepository struct{}
func NewTodoRepository(r *core.Registry) repositories.TodoRepository { return &todoRepository{} }
`,
	})
	src := generate(t, dir, "beego")
	if !strings.Contains(src, `return NewTodoRepository(r), nil`) {
		t.Errorf("want registry-style call\n%s", src)
	}
	if strings.Contains(src, "EntClientPlugin") {
		t.Errorf("beego must not emit EntClientPlugin\n%s", src)
	}
}

func TestReposUnsupportedParamFails(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"repositories/gorm/bad_repository.go": `package gorm
type badRepository struct{}
func NewBadRepository(cb func() int) *badRepository { return &badRepository{} }
`,
	})
	_, err := discoverRepositoryRegistrations(dir, "repositories", "db/client/ent", "gorm", "myapp", polaPkg)
	if err == nil {
		t.Fatal("want error for unsupported constructor param, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("want unsupported error, got %v", err)
	}
}

func TestReposMixedRegistryAndDepRejected(t *testing.T) {
	dir := setupProject(t, map[string]string{
		"repositories/gorm/todo_repository.go": `package gorm
import (
	"github.com/polagonow/pola/core"
	"gorm.io/gorm"
	"myapp/repositories"
)
type todoRepository struct{}
func NewTodoRepository(r *core.Registry, db *gorm.DB) repositories.TodoRepository { return &todoRepository{} }
`,
	})
	_, err := discoverRepositoryRegistrations(dir, "repositories", "db/client/ent", "gorm", "myapp", polaPkg)
	if err == nil || !strings.Contains(err.Error(), "*core.Registry") {
		t.Fatalf("want error rejecting mixed signature, got %v", err)
	}
}

package ent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/modelgen"
)

func TestEntGenerator_BasicModel(t *testing.T) {
	def := &modelgen.ModelDef{
		Name: "User",
		Fields: []modelgen.Field{
			{Name: "name", Type: modelgen.FieldString},
			{Name: "age", Type: modelgen.FieldInt},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	outFile := filepath.Join(dir, "ent", "user.go")
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `field.String("name")`)
	mustContain(t, content, `field.Int("age")`)
	mustContain(t, content, "type User struct")
	mustNotContain(t, content, "Edges()")
	mustNotContain(t, content, "Indexes()")
}

func TestEntGenerator_WithIndexes(t *testing.T) {
	def := &modelgen.ModelDef{
		Name: "User",
		Fields: []modelgen.Field{
			{Name: "email", Type: modelgen.FieldString, Unique: true},
			{Name: "name", Type: modelgen.FieldString, Index: true},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ent", "user.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, "Indexes()")
	mustContain(t, content, `index.Fields("email").Unique()`)
	mustContain(t, content, `index.Fields("name")`)
}

func TestEntGenerator_WithReferences(t *testing.T) {
	def := &modelgen.ModelDef{
		Name: "Article",
		Fields: []modelgen.Field{
			{Name: "title", Type: modelgen.FieldString},
			{Name: "author", Type: modelgen.FieldReferences, RefIDType: modelgen.FieldInt},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ent", "article.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `field.Int("author_id")`)
	mustContain(t, content, "Edges()")
	mustContain(t, content, `edge.From("author", Author.Type)`)
}

func TestEntGenerator_WithUUIDReferences(t *testing.T) {
	def := &modelgen.ModelDef{
		Name: "Article",
		Fields: []modelgen.Field{
			{Name: "author", Type: modelgen.FieldReferences, RefIDType: modelgen.FieldUUID},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ent", "article.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `field.UUID("author_id")`)
}

func TestEntGenerator_WithPolymorphicReferences(t *testing.T) {
	def := &modelgen.ModelDef{
		Name: "Comment",
		Fields: []modelgen.Field{
			{Name: "body", Type: modelgen.FieldText},
			{Name: "commentable", Type: modelgen.FieldReferences, Polymorphic: true, Unique: true, RefIDType: modelgen.FieldInt},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ent", "comment.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `field.Int("commentable_id")`)
	mustContain(t, content, `field.String("commentable_type")`)
	mustContain(t, content, `index.Fields("commentable_id", "commentable_type").Unique()`)
	// Polymorphic refs should NOT produce edges.
	mustNotContain(t, content, "edge.From")
}

func mustContain(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("output does not contain %q\n---\n%s", substr, content)
	}
}

func mustNotContain(t *testing.T, content, substr string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("output should not contain %q\n---\n%s", substr, content)
	}
}

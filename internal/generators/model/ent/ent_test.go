package ent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

func TestEntGenerator_BasicModel(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "User",
		Fields: []schema.Field{
			{Name: "name", Type: schema.FieldString},
			{Name: "age", Type: schema.FieldInt},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	outFile := filepath.Join(dir, "schema", "user.go")
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
	def := &schema.ModelDefinition{
		Name: "User",
		Fields: []schema.Field{
			{Name: "email", Type: schema.FieldString, Unique: true},
			{Name: "name", Type: schema.FieldString, Index: true},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "schema", "user.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, "Indexes()")
	mustContain(t, content, `index.Fields("email").Unique()`)
	mustContain(t, content, `index.Fields("name")`)
}

// seedAuthorSchema writes a minimal Author ent schema so addReverseEdge (which
// patches the referenced model to add a reverse edge) has a file to patch.
func seedAuthorSchema(t *testing.T, dir string) {
	t.Helper()
	schemaDir := filepath.Join(dir, "schema")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}
	src := `package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Author struct{ ent.Schema }

func (Author) Fields() []ent.Field {
	return []ent.Field{field.String("name")}
}
`
	if err := os.WriteFile(filepath.Join(schemaDir, "author.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write author schema: %v", err)
	}
}

func TestEntGenerator_WithReferences(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldString},
			{Name: "author", Type: schema.FieldReferences, RefIDType: schema.FieldInt},
		},
	}

	dir := t.TempDir()
	seedAuthorSchema(t, dir)
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "schema", "article.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `field.Int("author_id")`)
	mustContain(t, content, "Edges()")
	mustContain(t, content, `edge.From("author", Author.Type)`)
}

func TestEntGenerator_WithUUIDReferences(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "author", Type: schema.FieldReferences, RefIDType: schema.FieldUUID},
		},
	}

	dir := t.TempDir()
	seedAuthorSchema(t, dir)
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "schema", "article.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `field.UUID("author_id")`)
}

func TestEntGenerator_WithPolymorphicReferences(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Comment",
		Fields: []schema.Field{
			{Name: "body", Type: schema.FieldText},
			{Name: "commentable", Type: schema.FieldReferences, Polymorphic: true, Unique: true, RefIDType: schema.FieldInt},
		},
	}

	dir := t.TempDir()
	gen := &EntGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "schema", "comment.go"))
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

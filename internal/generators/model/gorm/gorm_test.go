package gorm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

func TestGormGenerator_BasicModel(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "User",
		Fields: []schema.Field{
			{Name: "name", Type: schema.FieldString},
			{Name: "age", Type: schema.FieldInt},
		},
	}

	dir := t.TempDir()
	gen := &GormGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	outFile := filepath.Join(dir, "gorm", "user.go")
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, "gorm.Model")
	mustContain(t, content, `Name string`)
	mustContain(t, content, `Age int`)
	mustContain(t, content, `json:"name"`)
	mustContain(t, content, `json:"age"`)
}

func TestGormGenerator_WithIndexes(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "User",
		Fields: []schema.Field{
			{Name: "email", Type: schema.FieldString, Unique: true},
			{Name: "name", Type: schema.FieldString, Index: true},
		},
	}

	dir := t.TempDir()
	gen := &GormGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "gorm", "user.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, `gorm:"type:varchar(255);uniqueIndex"`)
	mustContain(t, content, `gorm:"type:varchar(255);index"`)
}

func TestGormGenerator_WithReferences(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldString},
			{Name: "author", Type: schema.FieldReferences, RefIDType: schema.FieldInt64},
		},
	}

	dir := t.TempDir()
	gen := &GormGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "gorm", "article.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, "AuthorID int64")
	mustContain(t, content, `Author Author`)
	mustContain(t, content, `gorm:"foreignKey:AuthorID"`)
}

func TestGormGenerator_WithPolymorphicReferences(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Comment",
		Fields: []schema.Field{
			{Name: "body", Type: schema.FieldText},
			{Name: "commentable", Type: schema.FieldReferences, Polymorphic: true, Unique: true, RefIDType: schema.FieldInt64},
		},
	}

	dir := t.TempDir()
	gen := &GormGenerator{}
	if err := gen.Generate(def, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "gorm", "comment.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	mustContain(t, content, "CommentableID int64")
	mustContain(t, content, "CommentableType string")
	mustContain(t, content, `uniqueIndex`)
	// Polymorphic should NOT have association struct field.
	mustNotContain(t, content, `foreignKey`)
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

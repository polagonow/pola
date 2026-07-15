package model

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/generators/model/canonical"
	"github.com/polagonow/pola/internal/generators/model/schema"
)

// seedModel writes a canonical model file into dir so ValidateReferences can
// resolve references against the single source of truth.
func seedModel(t *testing.T, dir string, def *schema.ModelDefinition) {
	t.Helper()
	if err := canonical.Generate(def, dir); err != nil {
		t.Fatalf("seed %s: %v", def.Name, err)
	}
}

func TestValidateReferences_ModelNotFound(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "models")
	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldString},
			{Name: "author", Type: schema.FieldReferences},
		},
	}
	err := ValidateReferences(def, dir)
	if err == nil {
		t.Fatal("expected error for missing referenced model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "pola generate model Author") {
		t.Errorf("error should suggest generating the model, got: %v", err)
	}
}

func TestValidateReferences_AutoID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "models")
	seedModel(t, dir, &schema.ModelDefinition{
		Name:   "Author",
		Fields: []schema.Field{{Name: "name", Type: schema.FieldString}},
	})
	def := &schema.ModelDefinition{
		Name:   "Article",
		Fields: []schema.Field{{Name: "author", Type: schema.FieldReferences}},
	}
	if err := ValidateReferences(def, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Auto-increment PK → empty RefIDType (uint foreign key).
	if got := def.Fields[0].RefIDType; got != "" {
		t.Errorf("RefIDType = %q, want empty (auto-increment)", got)
	}
}

func TestValidateReferences_UUIDID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "models")
	seedModel(t, dir, &schema.ModelDefinition{
		Name:   "Author",
		IDType: schema.FieldUUID,
		Fields: []schema.Field{{Name: "name", Type: schema.FieldString}},
	})
	def := &schema.ModelDefinition{
		Name:   "Article",
		Fields: []schema.Field{{Name: "author", Type: schema.FieldReferences}},
	}
	if err := ValidateReferences(def, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := def.Fields[0].RefIDType; got != schema.FieldUUID {
		t.Errorf("RefIDType = %q, want %q", got, schema.FieldUUID)
	}
}

func TestValidateReferences_PolymorphicSkipsExistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "models")
	def := &schema.ModelDefinition{
		Name:   "Comment",
		Fields: []schema.Field{{Name: "commentable", Type: schema.FieldReferences, Polymorphic: true}},
	}
	if err := ValidateReferences(def, dir); err != nil {
		t.Fatalf("unexpected error for polymorphic: %v", err)
	}
	if got := def.Fields[0].RefIDType; got != "" {
		t.Errorf("RefIDType = %q, want empty (default)", got)
	}
}

func TestValidateReferences_StorageBlobSkipsExistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "models")
	def := &schema.ModelDefinition{
		Name:   "User",
		Fields: []schema.Field{{Name: "avatar", Type: schema.FieldReferences, RefModel: "StorageBlob"}},
	}
	if err := ValidateReferences(def, dir); err != nil {
		t.Fatalf("StorageBlob reference should not require an existing model: %v", err)
	}
}

func TestValidateReferences_NoRefsNoError(t *testing.T) {
	def := &schema.ModelDefinition{
		Name:   "User",
		Fields: []schema.Field{{Name: "name", Type: schema.FieldString}},
	}
	if err := ValidateReferences(def, filepath.Join(t.TempDir(), "models")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

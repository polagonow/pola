package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

func TestValidateReferences_ModelNotFound(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "title", Type: schema.FieldString},
			{Name: "author", Type: schema.FieldReferences},
		},
	}

	dir := t.TempDir()
	err := ValidateReferences(def, dir, "ent")
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

func TestValidateReferences_EntDefaultID(t *testing.T) {
	dir := t.TempDir()
	entDir := filepath.Join(dir, "ent")
	os.MkdirAll(entDir, 0o755)

	// Write a minimal ent schema with default ID (no explicit id field).
	authorSchema := `package ent

type Author struct { ent.Schema }
func (Author) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
	}
}
`
	os.WriteFile(filepath.Join(entDir, "author.go"), []byte(authorSchema), 0o644)

	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "author", Type: schema.FieldReferences},
		},
	}

	err := ValidateReferences(def, dir, "ent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ent default ID type is int.
	if def.Fields[0].RefIDType != schema.FieldInt {
		t.Errorf("RefIDType = %q, want %q", def.Fields[0].RefIDType, schema.FieldInt)
	}
}

func TestValidateReferences_EntUUIDID(t *testing.T) {
	dir := t.TempDir()
	entDir := filepath.Join(dir, "ent")
	os.MkdirAll(entDir, 0o755)

	// Write an ent schema with UUID ID.
	authorSchema := `package ent

type Author struct { ent.Schema }
func (Author) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}),
		field.String("name"),
	}
}
`
	os.WriteFile(filepath.Join(entDir, "author.go"), []byte(authorSchema), 0o644)

	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "author", Type: schema.FieldReferences},
		},
	}

	err := ValidateReferences(def, dir, "ent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if def.Fields[0].RefIDType != schema.FieldUUID {
		t.Errorf("RefIDType = %q, want %q", def.Fields[0].RefIDType, schema.FieldUUID)
	}
}

func TestValidateReferences_GormDefaultID(t *testing.T) {
	dir := t.TempDir()
	gormDir := filepath.Join(dir, "gorm")
	os.MkdirAll(gormDir, 0o755)

	authorModel := `package gorm

type Author struct {
	gorm.Model
	Name string
}
`
	os.WriteFile(filepath.Join(gormDir, "author.go"), []byte(authorModel), 0o644)

	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "author", Type: schema.FieldReferences},
		},
	}

	err := ValidateReferences(def, dir, "gorm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// gorm.Model uses uint → mapped to int64.
	if def.Fields[0].RefIDType != schema.FieldInt64 {
		t.Errorf("RefIDType = %q, want %q", def.Fields[0].RefIDType, schema.FieldInt64)
	}
}

func TestValidateReferences_GormCustomID(t *testing.T) {
	dir := t.TempDir()
	gormDir := filepath.Join(dir, "gorm")
	os.MkdirAll(gormDir, 0o755)

	authorModel := `package gorm

type Author struct {
	ID   uuid.UUID
	Name string
}
`
	os.WriteFile(filepath.Join(gormDir, "author.go"), []byte(authorModel), 0o644)

	def := &schema.ModelDefinition{
		Name: "Article",
		Fields: []schema.Field{
			{Name: "author", Type: schema.FieldReferences},
		},
	}

	err := ValidateReferences(def, dir, "gorm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if def.Fields[0].RefIDType != schema.FieldUUID {
		t.Errorf("RefIDType = %q, want %q", def.Fields[0].RefIDType, schema.FieldUUID)
	}
}

func TestValidateReferences_PolymorphicSkipsExistence(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "Comment",
		Fields: []schema.Field{
			{Name: "commentable", Type: schema.FieldReferences, Polymorphic: true},
		},
	}

	dir := t.TempDir()
	// No models exist — polymorphic should not error.
	err := ValidateReferences(def, dir, "ent")
	if err != nil {
		t.Fatalf("unexpected error for polymorphic: %v", err)
	}

	// Should get default ID type for ent.
	if def.Fields[0].RefIDType != schema.FieldInt {
		t.Errorf("RefIDType = %q, want %q", def.Fields[0].RefIDType, schema.FieldInt)
	}
}

func TestValidateReferences_NoRefsNoError(t *testing.T) {
	def := &schema.ModelDefinition{
		Name: "User",
		Fields: []schema.Field{
			{Name: "name", Type: schema.FieldString},
		},
	}

	err := ValidateReferences(def, t.TempDir(), "ent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

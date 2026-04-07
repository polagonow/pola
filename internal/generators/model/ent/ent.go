// Package ent implements the Ent ORM model generator for pola.
package ent

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

//go:embed _templates/*
var templates embed.FS

var schemaTmpl = template.Must(
	template.New("ent_schema.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/ent_schema.tmpl"),
)

// EntGenerator generates Ent schema files.
type EntGenerator struct{}

func (g *EntGenerator) Name() string { return "ent" }

func (g *EntGenerator) Generate(def *schema.ModelDefinition, outDir string) error {
	dir := filepath.Join(outDir, "schema")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	filename := schema.SnakeCase(def.Name) + ".go"
	filePath := filepath.Join(dir, filename)

	data := buildEntData(def)

	var buf strings.Builder
	if err := schemaTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute ent template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	return nil
}

type entData struct {
	PackageName   string
	Name          string
	Fields        []entField
	Indexes       []string
	Edges         []string
	HasIndexes    bool
	HasEdges      bool
	HasUUIDImport bool
}

type entField struct {
	EntField string
}

func buildEntData(def *schema.ModelDefinition) entData {
	data := entData{
		PackageName: "schema",
		Name:        def.Name,
	}

	if def.HasUUIDPrimaryKey() {
		data.Fields = append(data.Fields, entField{
			EntField: `field.String("id").DefaultFunc(func() string { return uuid.New().String() })`,
		})
		data.HasUUIDImport = true
	}

	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences {
			// Add FK field(s) with type matching the referenced model's ID.
			fkEntType := entTypeName(f.RefIDType)
			if fkEntType == "" {
				fkEntType = "Int" // ent default
			}
			data.Fields = append(data.Fields, entField{
				EntField: entFieldCall(fkEntType, f.Name+"_id", f.Optional),
			})
			if f.Polymorphic {
				data.Fields = append(data.Fields, entField{
					EntField: entFieldCall("String", f.Name+"_type", f.Optional),
				})
			}

			// Index for polymorphic references.
			if f.Polymorphic && (f.Index || f.Unique) {
				idx := fmt.Sprintf("index.Fields(%q, %q)", f.Name+"_id", f.Name+"_type")
				if f.Unique {
					idx += ".Unique()"
				}
				data.Indexes = append(data.Indexes, idx)
			} else if f.Index || f.Unique {
				idx := fmt.Sprintf("index.Fields(%q)", f.Name+"_id")
				if f.Unique {
					idx += ".Unique()"
				}
				data.Indexes = append(data.Indexes, idx)
			}

			// Edge for non-polymorphic references.
			if !f.Polymorphic {
				edge := fmt.Sprintf(
					"edge.From(%q, %s.Type).Ref(%q).Field(%q).Unique().Required()",
					f.Name,
					schema.PascalCase(f.Name),
					schema.Pluralize(schema.SnakeCase(def.Name)),
					f.Name+"_id",
				)
				data.Edges = append(data.Edges, edge)
			}
		} else {
			call := entFieldCall(entTypeName(f.Type), f.Name, f.Optional)
			if f.Limit > 0 {
				call += fmt.Sprintf(".MaxLen(%d)", f.Limit)
			}
			data.Fields = append(data.Fields, entField{
				EntField: call,
			})

			if f.Index || f.Unique {
				idx := fmt.Sprintf("index.Fields(%q)", f.Name)
				if f.Unique {
					idx += ".Unique()"
				}
				data.Indexes = append(data.Indexes, idx)
			}
		}
	}

	data.HasIndexes = len(data.Indexes) > 0
	data.HasEdges = len(data.Edges) > 0

	return data
}

func entTypeName(ft schema.FieldType) string {
	switch ft {
	case schema.FieldString:
		return "String"
	case schema.FieldText:
		return "Text"
	case schema.FieldInt:
		return "Int"
	case schema.FieldInt64:
		return "Int"
	case schema.FieldFloat:
		return "Float"
	case schema.FieldBool:
		return "Bool"
	case schema.FieldTime:
		return "Time"
	case schema.FieldUUID:
		return "UUID"
	case schema.FieldBytes:
		return "Bytes"
	case schema.FieldJSON:
		return "JSON"
	default:
		return "String"
	}
}

func entFieldCall(typeName, fieldName string, optional bool) string {
	s := fmt.Sprintf("field.%s(%q)", typeName, fieldName)
	if optional {
		s += ".Optional()"
	}
	return s
}

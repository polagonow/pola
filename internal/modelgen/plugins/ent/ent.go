// Package ent implements the Ent ORM model generator for pola.
package ent

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/modelgen"
)

//go:embed _templates/*
var templates embed.FS

var schemaTmpl = template.Must(
	template.New("ent_schema.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/ent_schema.tmpl"),
)

// EntGenerator generates Ent schema files.
type EntGenerator struct{}

func (g *EntGenerator) Name() string { return "ent" }

func (g *EntGenerator) Generate(def *modelgen.ModelDef, outDir string) error {
	dir := filepath.Join(outDir, "ent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	filename := modelgen.SnakeCase(def.Name) + ".go"
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
	PackageName string
	Name        string
	Fields      []entField
	Indexes     []string
	Edges       []string
	HasIndexes  bool
	HasEdges    bool
}

type entField struct {
	EntField string
}

func buildEntData(def *modelgen.ModelDef) entData {
	data := entData{
		PackageName: "ent",
		Name:        def.Name,
	}

	for _, f := range def.Fields {
		if f.Type == modelgen.FieldReferences {
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
					modelgen.PascalCase(f.Name),
					modelgen.Pluralize(modelgen.SnakeCase(def.Name)),
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

func entTypeName(ft modelgen.FieldType) string {
	switch ft {
	case modelgen.FieldString:
		return "String"
	case modelgen.FieldText:
		return "Text"
	case modelgen.FieldInt:
		return "Int"
	case modelgen.FieldInt64:
		return "Int"
	case modelgen.FieldFloat:
		return "Float"
	case modelgen.FieldBool:
		return "Bool"
	case modelgen.FieldTime:
		return "Time"
	case modelgen.FieldUUID:
		return "UUID"
	case modelgen.FieldBytes:
		return "Bytes"
	case modelgen.FieldJSON:
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

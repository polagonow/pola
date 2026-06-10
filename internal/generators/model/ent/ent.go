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

	// Add reverse edges on referenced models so ent codegen succeeds.
	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences && !f.Polymorphic {
			if err := addReverseEdge(outDir, def, f); err != nil {
				return fmt.Errorf("add reverse edge for %q: %w", f.Name, err)
			}
		}
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
					f.ReferencedModel(),
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
	if schema.ValidatorFieldTypes[ft] {
		return "String"
	}
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

// addReverseEdge patches the target model's schema file to add a reverse
// edge.To(...) for a non-polymorphic reference. This is required by ent
// which mandates both sides of a relation to be declared.
func addReverseEdge(outDir string, def *schema.ModelDefinition, f schema.Field) error {
	refModel := f.ReferencedModel()
	refFile := filepath.Join(outDir, "schema", schema.SnakeCase(refModel)+".go")

	data, err := os.ReadFile(refFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", refFile, err)
	}
	content := string(data)

	edgeName := schema.Pluralize(schema.SnakeCase(def.Name))
	toEdge := fmt.Sprintf("\t\tedge.To(%q, %s.Type),", edgeName, def.Name)

	// Check if this reverse edge already exists.
	if strings.Contains(content, fmt.Sprintf("edge.To(%q, %s.Type)", edgeName, def.Name)) {
		return nil
	}

	if strings.Contains(content, "Edges() []ent.Edge") {
		// Append to existing Edges method: insert before the closing "}"
		marker := "\t\tedge.To("
		if !strings.Contains(content, marker) {
			// Has Edges() but only From edges or empty — find the return slice
			marker = "return []ent.Edge{"
		}

		// Find last edge entry before closing and insert our new edge
		idx := strings.LastIndex(content, "\t}")
		edgesIdx := strings.Index(content, "Edges() []ent.Edge")
		if edgesIdx < 0 || idx < edgesIdx {
			return fmt.Errorf("could not find Edges() closing brace in %s", refFile)
		}
		// Find the closing of the return slice (the line with just "\t}" after "return []ent.Edge{")
		returnIdx := strings.Index(content[edgesIdx:], "return []ent.Edge{")
		if returnIdx < 0 {
			return fmt.Errorf("could not find return statement in Edges() in %s", refFile)
		}
		absReturnIdx := edgesIdx + returnIdx
		closingIdx := strings.Index(content[absReturnIdx:], "\t}")
		if closingIdx < 0 {
			return fmt.Errorf("could not find closing brace for return in Edges() in %s", refFile)
		}
		insertPos := absReturnIdx + closingIdx
		content = content[:insertPos] + toEdge + "\n" + content[insertPos:]
	} else {
		// No Edges method — add one before the final closing.
		edgesMethod := fmt.Sprintf(
			"\n// Edges of the %s.\nfunc (%s) Edges() []ent.Edge {\n\treturn []ent.Edge{\n%s\n\t}\n}\n",
			refModel, refModel, toEdge,
		)
		content = content + edgesMethod

		// Add edge import if missing.
		if !strings.Contains(content, `"entgo.io/ent/schema/edge"`) {
			content = strings.Replace(content,
				`"entgo.io/ent/schema/field"`,
				"\"entgo.io/ent/schema/field\"\n\t\"entgo.io/ent/schema/edge\"",
				1,
			)
		}
	}

	if err := os.WriteFile(refFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", refFile, err)
	}
	return nil
}

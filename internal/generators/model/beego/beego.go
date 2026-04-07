// Package beego implements the Beego ORM model generator for pola.
package beego

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

var modelTmpl = template.Must(
	template.New("beego_model.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/beego_model.tmpl"),
)

// BeegoGenerator generates Beego ORM model files.
type BeegoGenerator struct{}

func (g *BeegoGenerator) Name() string { return "beego" }

func (g *BeegoGenerator) Generate(def *schema.ModelDefinition, outDir string) error {
	dir := filepath.Join(outDir, "beego")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	filename := schema.SnakeCase(def.Name) + ".go"
	filePath := filepath.Join(dir, filename)

	data := buildBeegoData(def)

	var buf strings.Builder
	if err := modelTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute beego template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	return nil
}

type beegoData struct {
	PackageName string
	Name        string
	TableName   string
	Fields      []beegoField
	Imports     []string
	HasUUIDPK   bool
}

type beegoField struct {
	StructField string
}

func buildBeegoData(def *schema.ModelDefinition) beegoData {
	data := beegoData{
		PackageName: "beego",
		Name:        def.Name,
		TableName:   schema.SnakeCase(schema.Pluralize(def.Name)),
	}

	imports := map[string]bool{}

	if def.HasUUIDPrimaryKey() {
		data.HasUUIDPK = true
	}

	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences {
			// FK field with type matching the referenced model's ID.
			fkGoType := schema.GoType(f.RefIDType)
			if fkGoType == "" || fkGoType == "any" {
				fkGoType = "int" // beego default
			}
			ormTag := buildBeegoRefOrmTag(f)
			jsonName := schema.SnakeCase(f.Name) + "_id"
			tag := buildStructTag(ormTag, jsonName)
			data.Fields = append(data.Fields, beegoField{
				StructField: fmt.Sprintf("%s %s %s", schema.PascalCase(f.Name)+"Id", fkGoType, tag),
			})

			if f.Polymorphic {
				// Type field for polymorphic.
				ormTag := buildBeegoRefOrmTag(f)
				jsonName := schema.SnakeCase(f.Name) + "_type"
				tag := buildStructTag(ormTag, jsonName)
				data.Fields = append(data.Fields, beegoField{
					StructField: fmt.Sprintf("%s string %s", schema.PascalCase(f.Name)+"Type", tag),
				})
			} else {
				// Association pointer field.
				refName := schema.PascalCase(f.Name)
				jsonName := schema.SnakeCase(f.Name)
				data.Fields = append(data.Fields, beegoField{
					StructField: fmt.Sprintf("%s *%s `orm:\"rel(fk)\" json:\"%s\"`",
						refName, refName, jsonName),
				})
			}
		} else {
			goType := beegoGoType(f)
			ormTag := buildBeegoFieldOrmTag(f)
			jsonName := schema.SnakeCase(f.Name)

			// Track imports.
			switch f.Type {
			case schema.FieldTime:
				imports[`"time"`] = true
			}

			fieldType := goType
			if f.Optional {
				fieldType = "*" + goType
			}

			tag := buildStructTag(ormTag, jsonName)
			data.Fields = append(data.Fields, beegoField{
				StructField: fmt.Sprintf("%s %s %s", schema.PascalCase(f.Name), fieldType, tag),
			})
		}
	}

	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	return data
}

// beegoGoType returns the Go type for a beego model field.
func beegoGoType(f schema.Field) string {
	switch f.Type {
	case schema.FieldString:
		return "string"
	case schema.FieldText:
		return "string"
	case schema.FieldInt:
		return "int"
	case schema.FieldInt64:
		return "int64"
	case schema.FieldFloat:
		return "float64"
	case schema.FieldBool:
		return "bool"
	case schema.FieldTime:
		return "time.Time"
	case schema.FieldUUID:
		return "string" // beego has no native UUID; use string with size(36)
	case schema.FieldBytes:
		return "[]byte"
	case schema.FieldJSON:
		return "string" // beego has no native JSON column; stored as text
	default:
		return "string"
	}
}

// buildBeegoFieldOrmTag builds the orm:"..." tag value for a regular field.
func buildBeegoFieldOrmTag(f schema.Field) string {
	var parts []string

	switch f.Type {
	case schema.FieldString, schema.FieldUUID:
		if f.Limit > 0 {
			parts = append(parts, fmt.Sprintf("size(%d)", f.Limit))
		} else if f.Type == schema.FieldUUID {
			parts = append(parts, "size(36)")
		} else {
			parts = append(parts, "size(255)")
		}
	case schema.FieldText, schema.FieldJSON:
		parts = append(parts, "type(text)")
	case schema.FieldFloat:
		parts = append(parts, "digits(12);decimals(4)")
	case schema.FieldTime:
		parts = append(parts, "type(datetime)")
	}

	if f.Optional {
		parts = append(parts, "null")
	}

	if f.Index {
		parts = append(parts, "index")
	}
	if f.Unique {
		parts = append(parts, "unique")
	}

	return strings.Join(parts, ";")
}

// buildStructTag constructs the full struct tag string. If ormTag is empty,
// only the json tag is emitted.
func buildStructTag(ormTag, jsonName string) string {
	if ormTag != "" {
		return fmt.Sprintf("`orm:\"%s\" json:\"%s\"`", ormTag, jsonName)
	}
	return fmt.Sprintf("`json:\"%s\"`", jsonName)
}

// buildBeegoRefOrmTag builds the orm:"..." tag value for a reference FK/type field.
func buildBeegoRefOrmTag(f schema.Field) string {
	var parts []string

	if f.Index {
		parts = append(parts, "index")
	}
	if f.Unique {
		parts = append(parts, "unique")
	}

	return strings.Join(parts, ";")
}

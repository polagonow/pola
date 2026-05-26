// Package gorm implements the GORM model generator for pola.
package gorm

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
	template.New("gorm_model.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/gorm_model.tmpl"),
)

// GormGenerator generates GORM model files.
type GormGenerator struct{}

func (g *GormGenerator) Name() string { return "gorm" }

func (g *GormGenerator) Generate(def *schema.ModelDefinition, outDir string) error {
	dir := filepath.Join(outDir, "gorm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	filename := schema.SnakeCase(def.Name) + ".go"
	filePath := filepath.Join(dir, filename)

	data := buildGormData(def)

	var buf strings.Builder
	if err := modelTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute gorm template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	return nil
}

type gormData struct {
	PackageName string
	Name        string
	SnakeName   string
	Fields      []gormField
	Imports     []string
	HasUUIDPK   bool
}

type gormField struct {
	StructField string
}

func buildGormData(def *schema.ModelDefinition) gormData {
	data := gormData{
		PackageName: "gorm",
		Name:        def.Name,
		SnakeName:   schema.SnakeCase(def.Name),
	}

	imports := map[string]bool{
		`"gorm.io/gorm"`: true,
	}

	if def.HasUUIDPrimaryKey() {
		data.HasUUIDPK = true
		imports[`"time"`] = true
		imports[`"github.com/google/uuid"`] = true
	}

	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences {
			// FK field with type matching the referenced model's ID.
			fkGoType := schema.GoType(f.RefIDType)
			if fkGoType == "" || fkGoType == "any" {
				fkGoType = "int64" // fallback
			}
			tags := buildGormTags(f, true)
			data.Fields = append(data.Fields, gormField{
				StructField: fmt.Sprintf("%s %s %s", schema.PascalCase(f.Name)+"ID", fkGoType, tags),
			})
			// Track imports for FK type.
			if f.RefIDType == schema.FieldUUID {
				imports[`"github.com/google/uuid"`] = true
			}

			if f.Polymorphic {
				// Type field for polymorphic.
				typeTags := buildGormTags(f, false)
				data.Fields = append(data.Fields, gormField{
					StructField: fmt.Sprintf("%s string %s", schema.PascalCase(f.Name)+"Type", typeTags),
				})
			} else {
				// Association struct field.
				fieldName := schema.PascalCase(f.Name)
				refType := f.ReferencedModel()
				data.Fields = append(data.Fields, gormField{
					StructField: fmt.Sprintf(
						"%s %s `gorm:\"foreignKey:%sID\" json:\"%s\"`",
						fieldName, refType, fieldName, f.Name,
					),
				})
			}
		} else {
			goType := schema.GoType(f.Type)
			tags := buildGormFieldTags(f)

			// Track imports.
			switch f.Type {
			case schema.FieldTime:
				imports[`"time"`] = true
			case schema.FieldUUID:
				imports[`"github.com/google/uuid"`] = true
			case schema.FieldJSON:
				imports[`"encoding/json"`] = true
			}

			data.Fields = append(data.Fields, gormField{
				StructField: fmt.Sprintf("%s %s %s", schema.PascalCase(f.Name), goType, tags),
			})
		}
	}

	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	return data
}

func buildGormTags(f schema.Field, isIDField bool) string {
	var gormParts []string
	var jsonName string

	if isIDField {
		jsonName = f.Name + "_id"
	} else {
		jsonName = f.Name + "_type"
	}

	if f.Index {
		gormParts = append(gormParts, "index")
	}
	if f.Unique {
		gormParts = append(gormParts, "uniqueIndex")
	}

	if len(gormParts) > 0 {
		return fmt.Sprintf("`gorm:\"%s\" json:\"%s\"`", strings.Join(gormParts, ";"), jsonName)
	}
	return fmt.Sprintf("`json:\"%s\"`", jsonName)
}

func buildGormFieldTags(f schema.Field) string {
	jsonName := schema.SnakeCase(f.Name)
	var gormParts []string

	switch f.Type {
	case schema.FieldString, schema.FieldUUID:
		if f.Limit > 0 {
			gormParts = append(gormParts, fmt.Sprintf("type:varchar(%d)", f.Limit))
		} else {
			gormParts = append(gormParts, "type:varchar(255)")
		}
	case schema.FieldText:
		gormParts = append(gormParts, "type:text")
	case schema.FieldBytes:
		if f.Limit > 0 {
			gormParts = append(gormParts, fmt.Sprintf("size:%d", f.Limit))
		}
	}

	if f.Index {
		gormParts = append(gormParts, "index")
	}
	if f.Unique {
		gormParts = append(gormParts, "uniqueIndex")
	}

	if len(gormParts) > 0 {
		return fmt.Sprintf("`gorm:\"%s\" json:\"%s\"`", strings.Join(gormParts, ";"), jsonName)
	}
	return fmt.Sprintf("`json:\"%s\"`", jsonName)
}

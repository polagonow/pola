// Package gorm implements the GORM model generator for pola.
package gorm

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

var modelTmpl = template.Must(
	template.New("gorm_model.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/gorm_model.tmpl"),
)

// GormGenerator generates GORM model files.
type GormGenerator struct{}

func (g *GormGenerator) Name() string { return "gorm" }

func (g *GormGenerator) Generate(def *modelgen.ModelDef, outDir string) error {
	dir := filepath.Join(outDir, "gorm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	filename := modelgen.SnakeCase(def.Name) + ".go"
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
}

type gormField struct {
	StructField string
}

func buildGormData(def *modelgen.ModelDef) gormData {
	data := gormData{
		PackageName: "gorm",
		Name:        def.Name,
		SnakeName:   modelgen.SnakeCase(def.Name),
	}

	imports := map[string]bool{
		`"gorm.io/gorm"`: true,
	}

	for _, f := range def.Fields {
		if f.Type == modelgen.FieldReferences {
			// FK field with type matching the referenced model's ID.
			fkGoType := modelgen.GoType(f.RefIDType)
			if fkGoType == "" || fkGoType == "any" {
				fkGoType = "int64" // fallback
			}
			tags := buildGormTags(f, true)
			data.Fields = append(data.Fields, gormField{
				StructField: fmt.Sprintf("%s %s %s", modelgen.PascalCase(f.Name)+"ID", fkGoType, tags),
			})
			// Track imports for FK type.
			if f.RefIDType == modelgen.FieldUUID {
				imports[`"github.com/google/uuid"`] = true
			}

			if f.Polymorphic {
				// Type field for polymorphic.
				typeTags := buildGormTags(f, false)
				data.Fields = append(data.Fields, gormField{
					StructField: fmt.Sprintf("%s string %s", modelgen.PascalCase(f.Name)+"Type", typeTags),
				})
			} else {
				// Association struct field.
				refName := modelgen.PascalCase(f.Name)
				data.Fields = append(data.Fields, gormField{
					StructField: fmt.Sprintf(
						"%s %s `gorm:\"foreignKey:%sID\" json:\"%s\"`",
						refName, refName, refName, f.Name,
					),
				})
			}
		} else {
			goType := modelgen.GoType(f.Type)
			tags := buildGormFieldTags(f)

			// Track imports.
			switch f.Type {
			case modelgen.FieldTime:
				imports[`"time"`] = true
			case modelgen.FieldUUID:
				imports[`"github.com/google/uuid"`] = true
			case modelgen.FieldJSON:
				imports[`"encoding/json"`] = true
			}

			data.Fields = append(data.Fields, gormField{
				StructField: fmt.Sprintf("%s %s %s", modelgen.PascalCase(f.Name), goType, tags),
			})
		}
	}

	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}

	return data
}

func buildGormTags(f modelgen.Field, isIDField bool) string {
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

func buildGormFieldTags(f modelgen.Field) string {
	jsonName := modelgen.SnakeCase(f.Name)
	var gormParts []string

	switch f.Type {
	case modelgen.FieldString, modelgen.FieldUUID:
		if f.Limit > 0 {
			gormParts = append(gormParts, fmt.Sprintf("type:varchar(%d)", f.Limit))
		} else {
			gormParts = append(gormParts, "type:varchar(255)")
		}
	case modelgen.FieldText:
		gormParts = append(gormParts, "type:text")
	case modelgen.FieldBytes:
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

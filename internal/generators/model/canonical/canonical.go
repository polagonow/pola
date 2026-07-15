// Package canonical emits the single, ORM-neutral model struct that is the
// source of truth for both the database schema and the repository/transport
// entity. It writes one db/models/<snake>.go per model, carrying only pola:,
// json:, and validate: tags — never an ORM import or ORM struct tag. ORM-specific
// views (gorm/beego mirror structs, ent schemas) are derived from the pola: tags
// downstream at migration / codegen time.
package canonical

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

//go:embed _templates/*
var templates embed.FS

var modelTmpl = template.Must(
	template.New("model.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/model.tmpl"),
)

// Generate writes the neutral canonical struct for def into
// {outDir}/{snake_name}.go. The package name is derived from outDir's base name
// (e.g. db/models → package models).
func Generate(def *schema.ModelDefinition, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", outDir, err)
	}

	data := build(def, filepath.Base(outDir))

	var buf strings.Builder
	if err := modelTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute canonical model template: %w", err)
	}

	filePath := filepath.Join(outDir, schema.SnakeCase(def.Name)+".go")
	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	return nil
}

type modelData struct {
	PackageName string
	Name        string
	PluralSnake string
	Fields      []string // pre-rendered "Name GoType `tags`" lines
	Imports     []string
}

func build(def *schema.ModelDefinition, pkgName string) modelData {
	data := modelData{
		PackageName: pkgName,
		Name:        def.Name,
		PluralSnake: schema.SnakeCase(schema.Pluralize(def.Name)),
	}
	imports := map[string]bool{}

	// Primary key.
	if def.HasUUIDPrimaryKey() {
		data.Fields = append(data.Fields, "ID string `pola:\"pk:uuid\" json:\"id\"`")
	} else {
		data.Fields = append(data.Fields, "ID uint `pola:\"pk\" json:\"id\"`")
	}

	// User-defined columns.
	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences {
			data.Fields = append(data.Fields, referenceField(f))
			if f.Polymorphic {
				data.Fields = append(data.Fields, polymorphicTypeField(f))
			}
			continue
		}
		goType := schema.GoType(f.Type)
		switch f.Type {
		case schema.FieldTime:
			imports[`"time"`] = true
		case schema.FieldUUID:
			imports[`"github.com/google/uuid"`] = true
		case schema.FieldJSON:
			imports[`"encoding/json"`] = true
		}
		if f.Optional && !strings.HasPrefix(goType, "[]") {
			goType = "*" + goType
		}
		data.Fields = append(data.Fields, scalarField(f, goType))
	}

	// Framework-owned lifecycle columns (see repository package): plain types,
	// no ORM tags. The generic repository adapters stamp/soft-delete these.
	if def.HasTimestamps {
		imports[`"time"`] = true
		data.Fields = append(data.Fields,
			"CreatedAt time.Time `json:\"created_at\"`",
			"UpdatedAt time.Time `json:\"updated_at\"`",
		)
	}
	if def.SoftDeletes {
		imports[`"time"`] = true
		data.Fields = append(data.Fields, "DeletedAt *time.Time `pola:\"soft_delete\" json:\"deleted_at,omitempty\"`")
	}

	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}
	sort.Strings(data.Imports)
	return data
}

// scalarField renders a non-reference column: "Name GoType `pola:.. json:.. validate:..`".
func scalarField(f schema.Field, goType string) string {
	pola := []string{"type:" + string(f.Type)}
	if f.Limit > 0 {
		pola = append(pola, fmt.Sprintf("limit:%d", f.Limit))
	}
	if f.Index {
		pola = append(pola, "index")
	}
	if f.Unique {
		pola = append(pola, "uniq")
	}
	if f.Optional {
		pola = append(pola, "null")
	}
	json := schema.SnakeCase(f.Name)
	return fmt.Sprintf("%s %s `pola:%q json:%q validate:%q`",
		schema.PascalCase(f.Name), goType, strings.Join(pola, ";"), json, validTag(f))
}

// referenceField renders the foreign-key scalar column for a references field.
func referenceField(f schema.Field) string {
	goType := "uint"
	if f.RefIDType == schema.FieldUUID {
		goType = "string"
	}
	pola := []string{"references:" + f.RefModel}
	if f.Polymorphic {
		pola = append(pola, "polymorphic")
	}
	if f.Index {
		pola = append(pola, "index")
	}
	if f.Unique {
		pola = append(pola, "uniq")
	}
	if f.Optional {
		pola = append(pola, "null")
	}
	name := schema.PascalCase(f.Name) + "ID"
	json := schema.SnakeCase(f.Name) + "_id"
	return fmt.Sprintf("%s %s `pola:%q json:%q`", name, goType, strings.Join(pola, ";"), json)
}

// polymorphicTypeField renders the companion "<X>Type" discriminator column.
func polymorphicTypeField(f schema.Field) string {
	name := schema.PascalCase(f.Name) + "Type"
	json := schema.SnakeCase(f.Name) + "_type"
	return fmt.Sprintf("%s string `pola:%q json:%q`", name, "polymorphic_type", json)
}

// validatorTag maps schema field types to go-playground/validator tag fragments
// (the runtime uses go-playground/validator; see validation/validation.go).
var validatorTag = map[schema.FieldType]string{
	schema.FieldUUID:         "uuid4",
	schema.FieldEmail:        "email",
	schema.FieldURL:          "url",
	schema.FieldIP:           "ip",
	schema.FieldIPv4:         "ipv4",
	schema.FieldIPv6:         "ipv6",
	schema.FieldMAC:          "mac",
	schema.FieldAlpha:        "alpha",
	schema.FieldNumeric:      "numeric",
	schema.FieldAlphanumeric: "alphanum",
	schema.FieldHexColor:     "hexcolor",
	schema.FieldCreditCard:   "credit_card",
	schema.FieldBase64:       "base64",
	schema.FieldLatitude:     "latitude",
	schema.FieldLongitude:    "longitude",
	schema.FieldSemver:       "semver",
}

func validTag(f schema.Field) string {
	if f.Type == schema.FieldBool {
		return "-"
	}
	base := "required"
	if f.Optional {
		base = "omitempty"
	}
	if tag, ok := validatorTag[f.Type]; ok {
		return base + "," + tag
	}
	return base
}

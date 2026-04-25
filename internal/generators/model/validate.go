package model

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

// ValidateReferences checks that all referenced models exist in the output
// directory and resolves the FK ID type from the referenced model's generated
// source. It mutates def.Fields[i].RefIDType for each references field.
func ValidateReferences(def *schema.ModelDefinition, outDir, plugin string) error {
	// Ent schemas live under "schema/" subdirectory, other ORMs use their name.
	subDir := plugin
	if plugin == "ent" {
		subDir = "schema"
	}
	pluginDir := filepath.Join(outDir, subDir)

	for i := range def.Fields {
		f := &def.Fields[i]
		if f.Type != schema.FieldReferences {
			continue
		}

		if f.Polymorphic {
			// Polymorphic references: target is dynamic at runtime,
			// skip existence check, default FK type.
			f.RefIDType = defaultIDType(plugin)
			continue
		}

		// Non-polymorphic: the referenced model must exist.
		refModelName := f.ReferencedModel()
		refFile := filepath.Join(pluginDir, schema.SnakeCase(refModelName)+".go")
		if _, err := os.Stat(refFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(
					"model %q not found in %s; generate it first with:\n  pola generate model %s ...",
					refModelName, pluginDir, refModelName,
				)
			}
			return fmt.Errorf("check referenced model %q: %w", refModelName, err)
		}

		idType, err := parseModelIDType(refFile, plugin)
		if err != nil {
			return fmt.Errorf("resolve ID type for %q: %w", f.Name, err)
		}
		f.RefIDType = idType
	}

	return nil
}

// defaultIDType returns the default primary key type for a given ORM plugin
// when the actual type cannot be determined (e.g. polymorphic references).
func defaultIDType(plugin string) schema.FieldType {
	switch plugin {
	case "ent":
		return schema.FieldInt // ent defaults to int IDs
	case "gorm":
		return schema.FieldInt64 // gorm.Model uses uint (we use int64 as closest)
	case "beego":
		return schema.FieldInt // beego defaults to int auto-increment PK
	default:
		return schema.FieldInt64
	}
}

// parseModelIDType reads a generated model file and determines the ID/PK
// field type. Each plugin has its own conventions for ID fields.
func parseModelIDType(filePath, plugin string) (schema.FieldType, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filePath, err)
	}
	content := string(data)

	switch plugin {
	case "ent":
		return parseEntIDType(content), nil
	case "gorm":
		return parseGormIDType(content), nil
	case "beego":
		return parseBeegoIDType(content), nil
	default:
		return defaultIDType(plugin), nil
	}
}

// entIDPattern matches ent field.X("id"...) calls to detect custom ID types.
// e.g. field.UUID("id", uuid.UUID{}), field.Int("id"), field.String("id")
var entIDPattern = regexp.MustCompile(`field\.(\w+)\("id"`)

// parseEntIDType extracts the ID type from an ent schema.
// Ent defaults to int if no explicit id field is defined.
func parseEntIDType(content string) schema.FieldType {
	matches := entIDPattern.FindStringSubmatch(content)
	if matches == nil {
		return schema.FieldInt // ent default
	}
	return entTypeToFieldType(matches[1])
}

// gormIDPattern matches a custom ID field in a GORM struct.
// e.g. "ID uint64", "ID uuid.UUID", "ID int"
var gormIDPattern = regexp.MustCompile(`(?m)^\s+ID\s+(\S+)`)

// parseGormIDType extracts the ID type from a GORM model struct.
// If gorm.Model is embedded, default is uint (mapped to int64).
// If a custom ID field is found, use that type.
func parseGormIDType(content string) schema.FieldType {
	matches := gormIDPattern.FindStringSubmatch(content)
	if matches != nil {
		return goTypeToFieldType(matches[1])
	}
	// gorm.Model embeds ID uint
	if strings.Contains(content, "gorm.Model") {
		return schema.FieldInt64
	}
	return schema.FieldInt64
}

func entTypeToFieldType(entType string) schema.FieldType {
	switch entType {
	case "String":
		return schema.FieldString
	case "Int":
		return schema.FieldInt
	case "Int64":
		return schema.FieldInt64
	case "UUID":
		return schema.FieldUUID
	case "Float":
		return schema.FieldFloat
	case "Bool":
		return schema.FieldBool
	default:
		return schema.FieldInt
	}
}

// beegoIDPattern matches the Id field in a Beego model struct.
// e.g. "Id int", "Id int64", "Id string"
var beegoIDPattern = regexp.MustCompile(`(?m)^\s+Id\s+(\S+)`)

// parseBeegoIDType extracts the ID type from a Beego model struct.
// Beego convention is `Id int` with `orm:"auto;pk"`.
func parseBeegoIDType(content string) schema.FieldType {
	matches := beegoIDPattern.FindStringSubmatch(content)
	if matches != nil {
		return goTypeToFieldType(matches[1])
	}
	return schema.FieldInt // beego default
}

func goTypeToFieldType(goType string) schema.FieldType {
	switch goType {
	case "string":
		return schema.FieldString
	case "int":
		return schema.FieldInt
	case "int64":
		return schema.FieldInt64
	case "uint", "uint64":
		return schema.FieldInt64
	case "float64":
		return schema.FieldFloat
	case "uuid.UUID":
		return schema.FieldUUID
	default:
		return schema.FieldInt64
	}
}

package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

// sizedFieldTypes are types that accept a numeric {N} limit option.
var sizedFieldTypes = map[schema.FieldType]bool{
	schema.FieldString: true,
	schema.FieldText:   true,
	schema.FieldBytes:  true,
}

// ParseArgs parses CLI arguments into a ModelDefinition.
// args[0] is the model name, args[1:] are field:type{opts}:modifier specs.
//
//	ParseArgs([]string{"Article", "title:string:index", "body:text", "author:references"})
func ParseArgs(args []string) (*schema.ModelDefinition, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("model name is required")
	}

	name := schema.PascalCase(args[0])
	if name == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	def := &schema.ModelDefinition{Name: name}

	for _, arg := range args[1:] {
		f, err := parseField(arg)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", arg, err)
		}
		if f.Name == "id" {
			if f.Type != schema.FieldUUID {
				return nil, fmt.Errorf("field \"id\": only uuid type is supported for custom primary keys")
			}
			def.IDType = f.Type
			continue
		}
		def.Fields = append(def.Fields, f)
	}

	return def, nil
}

// parseField parses a single field spec like "title:string:index" or
// "category:references{polymorphic}:uniq".
func parseField(spec string) (schema.Field, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return schema.Field{}, fmt.Errorf("expected field:type, got %q", spec)
	}

	name := parts[0]
	if name == "" {
		return schema.Field{}, fmt.Errorf("field name cannot be empty")
	}

	// Parse type and options: "references{polymorphic}" → type="references", opt="polymorphic"
	typeStr := parts[1]
	var opts string
	if i := strings.IndexByte(typeStr, '{'); i >= 0 {
		j := strings.IndexByte(typeStr, '}')
		if j < 0 || j < i {
			return schema.Field{}, fmt.Errorf("unclosed { in type %q", typeStr)
		}
		opts = typeStr[i+1 : j]
		typeStr = typeStr[:i] + typeStr[j+1:]
	}

	// Check for optional marker.
	optional := false
	if strings.HasSuffix(typeStr, "?") {
		optional = true
		typeStr = strings.TrimSuffix(typeStr, "?")
	}

	ft := schema.FieldType(typeStr)
	if !schema.ValidFieldTypes[ft] {
		return schema.Field{}, fmt.Errorf("unknown type %q; valid types: string, int, int64, float, bool, time, uuid, text, bytes, json, references", typeStr)
	}

	// Validate options: numeric {N} for sized types, "polymorphic" for references.
	polymorphic := false
	limit := 0
	if opts != "" {
		if n, err := strconv.Atoi(opts); err == nil {
			// Numeric limit, e.g. string{255}.
			if !sizedFieldTypes[ft] {
				return schema.Field{}, fmt.Errorf("{%s} size limit is not valid on %s type; valid on: string, text, bytes", opts, ft)
			}
			if n <= 0 {
				return schema.Field{}, fmt.Errorf("{%s} size limit must be a positive integer", opts)
			}
			limit = n
		} else {
			// Keyword option.
			if ft != schema.FieldReferences {
				return schema.Field{}, fmt.Errorf("{%s} option is only valid on references type", opts)
			}
			if opts != "polymorphic" {
				return schema.Field{}, fmt.Errorf("unknown option {%s}; valid options: polymorphic", opts)
			}
			polymorphic = true
		}
	}

	// Parse modifiers.
	var index, unique bool
	for _, mod := range parts[2:] {
		switch mod {
		case "index":
			index = true
		case "uniq":
			unique = true
		default:
			return schema.Field{}, fmt.Errorf("unknown modifier %q; valid modifiers: index, uniq", mod)
		}
	}

	return schema.Field{
		Name:        name,
		Type:        ft,
		Optional:    optional,
		Index:       index,
		Unique:      unique,
		Polymorphic: polymorphic,
		Limit:       limit,
	}, nil
}

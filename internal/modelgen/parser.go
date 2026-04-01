package modelgen

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// sizedFieldTypes are types that accept a numeric {N} limit option.
var sizedFieldTypes = map[FieldType]bool{
	FieldString: true,
	FieldText:   true,
	FieldBytes:  true,
}

// ParseArgs parses CLI arguments into a ModelDef.
// args[0] is the model name, args[1:] are field:type{opts}:modifier specs.
//
//	ParseArgs([]string{"Article", "title:string:index", "body:text", "author:references"})
func ParseArgs(args []string) (*ModelDef, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("model name is required")
	}

	name := PascalCase(args[0])
	if name == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	def := &ModelDef{Name: name}

	for _, arg := range args[1:] {
		f, err := parseField(arg)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", arg, err)
		}
		def.Fields = append(def.Fields, f)
	}

	return def, nil
}

// parseField parses a single field spec like "title:string:index" or
// "category:references{polymorphic}:uniq".
func parseField(spec string) (Field, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return Field{}, fmt.Errorf("expected field:type, got %q", spec)
	}

	name := parts[0]
	if name == "" {
		return Field{}, fmt.Errorf("field name cannot be empty")
	}

	// Parse type and options: "references{polymorphic}" → type="references", opt="polymorphic"
	typeStr := parts[1]
	var opts string
	if i := strings.IndexByte(typeStr, '{'); i >= 0 {
		j := strings.IndexByte(typeStr, '}')
		if j < 0 || j < i {
			return Field{}, fmt.Errorf("unclosed { in type %q", typeStr)
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

	ft := FieldType(typeStr)
	if !validFieldTypes[ft] {
		return Field{}, fmt.Errorf("unknown type %q; valid types: string, int, int64, float, bool, time, uuid, text, bytes, json, references", typeStr)
	}

	// Validate options: numeric {N} for sized types, "polymorphic" for references.
	polymorphic := false
	limit := 0
	if opts != "" {
		if n, err := strconv.Atoi(opts); err == nil {
			// Numeric limit, e.g. string{255}.
			if !sizedFieldTypes[ft] {
				return Field{}, fmt.Errorf("{%s} size limit is not valid on %s type; valid on: string, text, bytes", opts, ft)
			}
			if n <= 0 {
				return Field{}, fmt.Errorf("{%s} size limit must be a positive integer", opts)
			}
			limit = n
		} else {
			// Keyword option.
			if ft != FieldReferences {
				return Field{}, fmt.Errorf("{%s} option is only valid on references type", opts)
			}
			if opts != "polymorphic" {
				return Field{}, fmt.Errorf("unknown option {%s}; valid options: polymorphic", opts)
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
			return Field{}, fmt.Errorf("unknown modifier %q; valid modifiers: index, uniq", mod)
		}
	}

	return Field{
		Name:        name,
		Type:        ft,
		Optional:    optional,
		Index:       index,
		Unique:      unique,
		Polymorphic: polymorphic,
		Limit:       limit,
	}, nil
}

// PascalCase converts a string to PascalCase.
// "article" → "Article", "blog_post" → "BlogPost", "Article" → "Article"
func PascalCase(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	upper := true
	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Pluralize returns a naive English plural of s.
// "article" → "articles", "category" → "categories", "bus" → "buses"
func Pluralize(s string) string {
	if s == "" {
		return s
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") ||
		strings.HasSuffix(s, "sh") || strings.HasSuffix(s, "ch") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		c := s[len(s)-2]
		if c != 'a' && c != 'e' && c != 'i' && c != 'o' && c != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}

// SnakeCase converts a PascalCase or camelCase string to snake_case.
// "Article" → "article", "BlogPost" → "blog_post"
func SnakeCase(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

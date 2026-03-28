// Package codegen generates Go bridge code and TypeScript declarations
// from action structs in a project's actions/ directory.
package codegen

import (
	"fmt"
	"go/types"
	"strings"
	"unicode"
)

// TSType converts a Go types.Type to its TypeScript representation.
// namedTypes accumulates any named struct types encountered so they
// can be emitted as TS interfaces.
func TSType(t types.Type, namedTypes map[string]*types.Struct) string {
	switch u := t.(type) {
	case *types.Basic:
		return tsBasic(u)
	case *types.Pointer:
		inner := TSType(u.Elem(), namedTypes)
		return inner + " | null"
	case *types.Slice:
		inner := TSType(u.Elem(), namedTypes)
		return inner + "[]"
	case *types.Array:
		inner := TSType(u.Elem(), namedTypes)
		return inner + "[]"
	case *types.Map:
		key := TSType(u.Key(), namedTypes)
		val := TSType(u.Elem(), namedTypes)
		return fmt.Sprintf("Record<%s, %s>", key, val)
	case *types.Named:
		obj := u.Obj()
		// time.Time → string (JSON serializes as ISO string)
		if obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "Time" {
			return "string"
		}
		// error → string
		if obj.Name() == "error" && obj.Pkg() == nil {
			return "string"
		}
		// Named struct → emit as TS interface
		if st, ok := u.Underlying().(*types.Struct); ok {
			name := obj.Name()
			if _, exists := namedTypes[name]; !exists {
				// Register before recursing to handle cycles
				namedTypes[name] = st
			}
			return name
		}
		// Named non-struct (e.g. type ID string) → resolve underlying
		return TSType(u.Underlying(), namedTypes)
	case *types.Interface:
		return "unknown"
	default:
		return "unknown"
	}
}

func tsBasic(b *types.Basic) string {
	switch b.Kind() {
	case types.String, types.UntypedString:
		return "string"
	case types.Bool, types.UntypedBool:
		return "boolean"
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Float32, types.Float64,
		types.UntypedInt, types.UntypedFloat:
		return "number"
	default:
		return "unknown"
	}
}

// CamelCase converts a PascalCase Go name to camelCase for JS.
func CamelCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	// Lowercase leading uppercase run (e.g. "HTMLParser" → "htmlParser")
	i := 0
	for i < len(runes) && unicode.IsUpper(runes[i]) {
		i++
	}
	if i == 0 {
		return s
	}
	if i == 1 {
		runes[0] = unicode.ToLower(runes[0])
	} else if i == len(runes) {
		// All uppercase: lowercase everything
		for j := range runes {
			runes[j] = unicode.ToLower(runes[j])
		}
	} else {
		// Lowercase all but the last uppercase (it starts the next word)
		for j := 0; j < i-1; j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
	}
	return string(runes)
}

// StructToTSFields converts a Go struct type to TypeScript interface fields.
// Returns lines like: "  fieldName: type;"
func StructToTSFields(st *types.Struct, namedTypes map[string]*types.Struct) []string {
	var lines []string
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		tag := st.Tag(i)
		jsonName := jsonFieldName(f.Name(), tag)
		if jsonName == "-" {
			continue
		}
		tsType := TSType(f.Type(), namedTypes)
		// Check if omitempty — make optional
		optional := ""
		if strings.Contains(tag, "omitempty") {
			optional = "?"
		}
		lines = append(lines, fmt.Sprintf("  %s%s: %s;", jsonName, optional, tsType))
	}
	return lines
}

// jsonFieldName extracts the JSON field name from a struct tag.
// Falls back to camelCase of the Go field name.
func jsonFieldName(goName, tag string) string {
	// Parse json tag from raw tag string
	jsonTag := ""
	for _, part := range strings.Split(tag, " ") {
		part = strings.Trim(part, "`")
		if strings.HasPrefix(part, `json:"`) {
			jsonTag = strings.TrimPrefix(part, `json:"`)
			jsonTag = strings.TrimSuffix(jsonTag, `"`)
			break
		}
	}
	if jsonTag == "" {
		return CamelCase(goName)
	}
	name := strings.Split(jsonTag, ",")[0]
	if name == "" {
		return CamelCase(goName)
	}
	return name
}

// Package actionbridge generates Go bridge code and TypeScript declarations
// from action structs in a project's actions/ directory.
package actionbridge

import (
	"fmt"
	"go/types"
	"unicode"
)

// TSType converts a Go types.Type to its TypeScript representation.
// Struct types are referenced by name — their interface definitions
// are emitted by emitNamedTypes.
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
		if obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "Time" {
			return "string"
		}
		if obj.Name() == "error" && obj.Pkg() == nil {
			return "string"
		}
		if st, ok := u.Underlying().(*types.Struct); ok {
			name := obj.Name()
			if _, exists := namedTypes[name]; !exists {
				namedTypes[name] = st
			}
			return name
		}
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
		for j := range runes {
			runes[j] = unicode.ToLower(runes[j])
		}
	} else {
		for j := 0; j < i-1; j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
	}
	return string(runes)
}

package schema

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// ParseModelsDir parses every non-test .go file in dir into ModelDefinitions,
// keyed by PascalCase model name. It is used to resolve cross-model references
// (a field's foreign-key ID type) from the single source of truth after a human
// edits the canonical structs.
func ParseModelsDir(dir string) (map[string]*ModelDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string]*ModelDefinition{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		def, err := ParseModelFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if def != nil {
			out[def.Name] = def
		}
	}
	return out, nil
}

// ParseModelFile reads the first exported struct in a canonical db/models file
// back into a ModelDefinition. It is the inverse of the canonical emitter: the
// pair must round-trip so that editing a struct and generating from the CLI
// converge on the same IR. Returns (nil, nil) when the file declares no exported
// struct.
func ParseModelFile(path string) (*ModelDefinition, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !ts.Name.IsExported() {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			return structToDefinition(ts.Name.Name, st)
		}
	}
	return nil, nil
}

// structToDefinition reconstructs a ModelDefinition from a canonical struct. PK,
// timestamps, and the soft-delete column are struct-level conventions recovered
// by field name + tag; only user-defined columns land in Fields.
func structToDefinition(name string, st *ast.StructType) (*ModelDefinition, error) {
	def := &ModelDefinition{Name: name}

	for _, astField := range st.Fields.List {
		if len(astField.Names) == 0 {
			continue // embedded field — canonical models never embed
		}
		fieldName := astField.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue
		}

		goType := exprString(astField.Type)
		pola := polaTag(astField.Tag)
		tokens := parsePolaTokens(pola)

		// Skip explicitly-ignored and association fields.
		if _, skip := tokens["-"]; skip {
			continue
		}
		if _, ok := tokens["belongs_to"]; ok {
			continue
		}

		// Primary key (by tag or by the conventional field name "ID").
		if _, isPK := tokens["pk"]; isPK || strings.EqualFold(fieldName, "id") {
			if tokens["pk"] == "uuid" || strings.TrimPrefix(goType, "*") == "string" {
				def.IDType = FieldUUID
			}
			continue
		}

		// Timestamp / soft-delete conventions.
		if _, ok := tokens["soft_delete"]; ok || fieldName == "DeletedAt" {
			def.SoftDeletes = true
			continue
		}
		if fieldName == "CreatedAt" || fieldName == "UpdatedAt" {
			def.HasTimestamps = true
			continue
		}

		f, ok := tokensToField(fieldName, goType, tokens)
		if !ok {
			continue
		}
		def.Fields = append(def.Fields, f)
	}

	return def, nil
}

// tokensToField builds a Field from a struct field's name, Go type, and pola
// tokens. Returns ok=false for fields that are not model columns.
func tokensToField(fieldName, goType string, tokens map[string]string) (Field, bool) {
	optional := strings.HasPrefix(goType, "*")
	if _, ok := tokens["null"]; ok {
		optional = true
	}

	// References: named "<X>ID", carrying references:Model (empty target =
	// polymorphic). The FK's own Go type recovers the referenced PK type.
	if ref, ok := tokens["references"]; ok {
		specName := SnakeCase(strings.TrimSuffix(fieldName, "ID"))
		f := Field{
			Name:        specName,
			Type:        FieldReferences,
			Optional:    optional,
			Index:       hasFlag(tokens, "index"),
			Unique:      hasFlag(tokens, "uniq"),
			Polymorphic: hasFlag(tokens, "polymorphic"),
			RefModel:    ref,
		}
		if strings.TrimPrefix(goType, "*") == "string" {
			f.RefIDType = FieldUUID
		}
		return f, true
	}

	// The polymorphic "<X>Type" companion column is derived, not a field.
	if hasFlag(tokens, "polymorphic_type") {
		return Field{}, false
	}

	typeToken := tokens["type"]
	if typeToken == "" {
		return Field{}, false // not a recognised column
	}
	ft := FieldType(typeToken)
	if !ValidFieldTypes[ft] {
		return Field{}, false
	}

	limit := 0
	if l := tokens["limit"]; l != "" {
		limit, _ = strconv.Atoi(l)
	}

	return Field{
		Name:     SnakeCase(fieldName),
		Type:     ft,
		Optional: optional,
		Index:    hasFlag(tokens, "index"),
		Unique:   hasFlag(tokens, "uniq"),
		Limit:    limit,
	}, true
}

func hasFlag(tokens map[string]string, key string) bool {
	_, ok := tokens[key]
	return ok
}

// polaTag extracts the `pola:"..."` value from a struct field's raw tag literal.
func polaTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	unquoted, err := strconv.Unquote(tag.Value)
	if err != nil {
		return ""
	}
	return reflect.StructTag(unquoted).Get("pola")
}

// parsePolaTokens splits a pola tag value into a key→value map. Bare tokens map
// to an empty value: "type:string;limit:255;index" →
// {"type":"string","limit":"255","index":""}.
func parsePolaTokens(pola string) map[string]string {
	tokens := map[string]string{}
	for _, part := range strings.Split(pola, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if k, v, ok := strings.Cut(part, ":"); ok {
			tokens[k] = v
		} else {
			tokens[part] = ""
		}
	}
	return tokens
}

// exprString renders a field type AST node to its Go source string, covering the
// shapes canonical models use: identifiers (int, string), qualified names
// (time.Time, uuid.UUID, json.RawMessage), pointers, and []byte.
func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return "[...]" + exprString(t.Elt)
	default:
		return ""
	}
}

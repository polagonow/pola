package actionbridge

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

type goTemplData struct {
	PackageName        string
	PolaPackage        string
	Actions            []ActionDef
	ConstructorImports []string // unique import paths needed by constructors
}

func (d goTemplData) HasAnyVars() bool {
	for _, a := range d.Actions {
		if a.HasVars {
			return true
		}
	}
	return false
}

func (d goTemplData) StructNameFor(m MethodDef) string {
	for _, a := range d.Actions {
		for _, am := range a.Methods {
			if am.GoName == m.GoName && am.JSName == m.JSName {
				return a.StructName
			}
		}
	}
	return ""
}

// GenerateGo produces the generated_bridge.go source code.
func GenerateGo(result *ParseResult, polaPackage string) ([]byte, error) {
	funcMap := template.FuncMap{
		"fieldName": func(structName string) string {
			return CamelCase(structName)
		},
		"goTypeShort": func(goType string) string {
			// Strip package path for types within the same package
			if idx := strings.LastIndex(goType, "/"); idx >= 0 {
				// Find the type name after the package
				rest := goType[idx+1:]
				if dotIdx := strings.Index(rest, "."); dotIdx >= 0 {
					return rest
				}
			}
			return goType
		},
		"castArg": func(i int, p ParamDef) string {
			return goArgCast(i, p)
		},
		"callArgs": func(params []ParamDef) string {
			args := make([]string, len(params))
			for i := range params {
				args[i] = fmt.Sprintf("p%d", i)
			}
			return strings.Join(args, ", ")
		},
	}

	tmpl, err := template.New("bridge").Funcs(funcMap).Parse(goTmpl)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	data := goTemplData{
		PackageName:        result.PackageName,
		PolaPackage:        polaPackage,
		Actions:            result.Actions,
		ConstructorImports: collectConstructorImports(result.Actions),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// collectConstructorImports returns deduplicated import paths needed by all action constructors.
func collectConstructorImports(actions []ActionDef) []string {
	seen := make(map[string]bool)
	for _, a := range actions {
		if a.Constructor == nil {
			continue
		}
		for _, p := range a.Constructor.Params {
			if p.TypePath != "" {
				seen[p.TypePath] = true
			}
		}
	}
	imports := make([]string, 0, len(seen))
	for imp := range seen {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

func goArgCast(i int, p ParamDef) string {
	gt := p.GoType
	// Handle basic types
	switch {
	case gt == "string":
		return fmt.Sprintf(`if s, ok := args[%d].(string); ok { p%d = s }`, i, i)
	case gt == "int" || gt == "int64":
		return fmt.Sprintf(`switch v := args[%d].(type) {
				case float64: p%d = %s(v)
				case int: p%d = %s(v)
				case int64: p%d = %s(v)
			}`, i, i, gt, i, gt, i, gt)
	case gt == "float64":
		return fmt.Sprintf(`if v, ok := args[%d].(float64); ok { p%d = v }`, i, i)
	case gt == "float32":
		return fmt.Sprintf(`if v, ok := args[%d].(float64); ok { p%d = float32(v) }`, i, i)
	case gt == "bool":
		return fmt.Sprintf(`if v, ok := args[%d].(bool); ok { p%d = v }`, i, i)
	default:
		// For complex types, try JSON round-trip
		return fmt.Sprintf(`if b, err := json.Marshal(args[%d]); err == nil { _ = json.Unmarshal(b, &p%d) }`, i, i)
	}
}

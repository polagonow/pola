package actionbridge

import (
	"bytes"
	"fmt"
	"go/types"
	"slices"
	"sort"
	"strings"
	"text/template"
)

type tsAction struct {
	StructName string
	Methods    []tsMethod
	HasVars    bool
}

type tsMethod struct {
	JSName     string
	ParamsList string
	ReturnType string
}

// GenerateTS produces the generated.ts TypeScript source with runtime code + types.
func GenerateTS(result *ParseResult) ([]byte, error) {
	var structInterfaces string
	if len(result.NamedTypes) > 0 {
		var buf bytes.Buffer
		emitNamedTypes(&buf, result.NamedTypes)
		structInterfaces = buf.String()
	}

	// Build action data for template.
	var actions []tsAction
	for _, action := range result.Actions {
		a := tsAction{
			StructName: action.StructName,
			HasVars:    action.HasVars,
		}
		for _, m := range action.Methods {
			params := make([]string, len(m.Params))
			for i, p := range m.Params {
				params[i] = fmt.Sprintf("%s: %s", p.Name, p.TSType)
			}
			returnType := "void"
			if m.HasReturn {
				returnType = m.ReturnTS
			}
			a.Methods = append(a.Methods, tsMethod{
				JSName:     m.JSName,
				ParamsList: strings.Join(params, ", "),
				ReturnType: returnType,
			})
		}
		actions = append(actions, a)
	}

	generatedTSTmpl, err := template.New("generated").Parse(tsTmpl)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	err = generatedTSTmpl.Execute(&buf, struct {
		StructInterfaces string
		Actions          []tsAction
	}{
		StructInterfaces: structInterfaces,
		Actions:          actions,
	})
	if err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

func emitNamedTypes(buf *bytes.Buffer, namedTypes map[string]*types.Struct) {
	emitted := map[string]bool{}
	queue := make([]string, 0, len(namedTypes))
	for name := range namedTypes {
		queue = append(queue, name)
	}
	sort.Strings(queue)

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if emitted[name] {
			continue
		}
		emitted[name] = true

		st := namedTypes[name]

		var embedded []string
		type fieldEntry struct {
			jsonName string
			tsType   string
			optional bool
		}
		var fields []fieldEntry

		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if !f.Exported() {
				continue
			}
			tag := st.Tag(i)

			if f.Anonymous() {
				if named, ok := f.Type().(*types.Named); ok {
					embName := named.Obj().Name()
					if underlying, ok := named.Underlying().(*types.Struct); ok {
						if _, exists := namedTypes[embName]; !exists {
							namedTypes[embName] = underlying
						}
						if !emitted[embName] {
							queue = append(queue, embName)
						}
					}
					embedded = append(embedded, embName)
				}
				continue
			}

			jsonName := jsonFieldName(f.Name(), tag)
			if jsonName == "-" {
				continue
			}

			tsType := TSType(f.Type(), namedTypes)

			for n := range namedTypes {
				if !emitted[n] && !slices.Contains(queue, n) {
					queue = append(queue, n)
				}
			}

			isOptional := strings.Contains(tag, "omitempty")
			fields = append(fields, fieldEntry{jsonName, tsType, isOptional})
		}

		if len(embedded) > 0 {
			buf.WriteString(fmt.Sprintf("export interface %s extends %s {\n", name, strings.Join(embedded, ", ")))
		} else {
			buf.WriteString(fmt.Sprintf("export interface %s {\n", name))
		}
		for _, f := range fields {
			opt := ""
			if f.optional {
				opt = "?"
			}
			buf.WriteString(fmt.Sprintf("  %s%s: %s;\n", f.jsonName, opt, f.tsType))
		}
		buf.WriteString("}\n\n")
	}
}

// jsonFieldName extracts the JSON field name from a struct tag.
func jsonFieldName(goName, tag string) string {
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

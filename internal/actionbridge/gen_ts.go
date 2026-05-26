package actionbridge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type tsGenData struct {
	PackagePath string
	StructNames []string
}

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
	// Generate struct interfaces via typescriptify if there are named types.
	var structInterfaces string
	if len(result.NamedTypes) > 0 {
		ts, err := runTypescriptify(result)
		if err != nil {
			// Fallback: use our own emitter.
			var buf bytes.Buffer
			emitNamedTypesFallback(&buf, result)
			structInterfaces = buf.String()
		} else {
			structInterfaces = ts
		}
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

// runTypescriptify generates a temporary Go program that uses typescriptify
// to convert action structs to TypeScript interfaces, compiles and runs it.
func runTypescriptify(result *ParseResult) (string, error) {
	// Collect struct names that need TS interfaces.
	var structNames []string
	for name := range result.NamedTypes {
		structNames = append(structNames, name)
	}
	if len(structNames) == 0 {
		return "", nil
	}

	// Create temp directory for the generator program.
	tmpDir, err := os.MkdirTemp("", "pola-tsgen-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Render the generator program.
	tmpl, err := template.New("tsgen").Parse(tsGenTmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := tsGenData{
		PackagePath: result.PackagePath,
		StructNames: structNames,
	}

	var progBuf bytes.Buffer
	if err := tmpl.Execute(&progBuf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	// Write the program.
	mainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainPath, progBuf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write temp program: %w", err)
	}

	// Write a go.mod that requires the user's module + typescriptify.
	goMod := fmt.Sprintf(`module pola-tsgen

go 1.25.0

require (
	%s v0.0.0
	github.com/tkrajina/typescriptify-golang-structs v0.2.0
)
`, result.PackagePath)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return "", fmt.Errorf("write go.mod: %w", err)
	}

	// Find the actions directory to set up a replace directive.
	actionsModRoot, actionsModPath, err := findModuleRoot(result.PackagePath)
	if err != nil {
		return "", fmt.Errorf("find module root: %w", err)
	}

	// Add replace directive for the user's module.
	goModReplace := fmt.Sprintf("\nreplace %s => %s\n", actionsModPath, actionsModRoot)
	f, err := os.OpenFile(filepath.Join(tmpDir, "go.mod"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	f.WriteString(goModReplace)
	f.Close()

	// Run go mod tidy to resolve dependencies.
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod tidy: %s\n%w", out, err)
	}

	// Run the generator.
	runCmd := exec.Command("go", "run", "main.go")
	runCmd.Dir = tmpDir
	var stdout, stderr bytes.Buffer
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr
	if err := runCmd.Run(); err != nil {
		return "", fmt.Errorf("run tsgen: %s\n%w", stderr.String(), err)
	}

	output := stdout.String()
	if strings.HasPrefix(output, "ERROR:") {
		return "", fmt.Errorf("typescriptify: %s", output)
	}

	return output, nil
}

// findModuleRoot walks up from the current directory to find the go.mod
// that contains the given package path, returning the directory and module path.
func findModuleRoot(pkgPath string) (string, string, error) {
	cmd := exec.Command("go", "list", "-m", "-json")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		dir, _ := os.Getwd()
		return dir, pkgPath, nil
	}

	output := out.String()
	modDir := extractJSONField(output, "Dir")
	modPath := extractJSONField(output, "Path")
	if modDir != "" && modPath != "" {
		return modDir, modPath, nil
	}

	dir, _ := os.Getwd()
	return dir, pkgPath, nil
}

func extractJSONField(json, field string) string {
	key := fmt.Sprintf(`"%s":`, field)
	idx := strings.Index(json, key)
	if idx < 0 {
		return ""
	}
	rest := json[idx+len(key):]
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// emitNamedTypesFallback is the fallback TS struct emitter used when
// typescriptify can't run (e.g. missing dependencies in the user's project).
func emitNamedTypesFallback(buf *bytes.Buffer, result *ParseResult) {
	emitted := map[string]bool{}
	for {
		progress := false
		for name, st := range result.NamedTypes {
			if emitted[name] {
				continue
			}
			emitted[name] = true
			progress = true

			buf.WriteString(fmt.Sprintf("export interface %s {\n", name))
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
				tsType := TSType(f.Type(), result.NamedTypes)
				optional := ""
				if strings.Contains(tag, "omitempty") {
					optional = "?"
				}
				buf.WriteString(fmt.Sprintf("  %s%s: %s;\n", jsonName, optional, tsType))
			}
			buf.WriteString("}\n\n")
		}
		if !progress {
			break
		}
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

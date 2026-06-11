// Package zod implements the Zod schema generator for the Pola CLI.
// It generates TypeScript Zod validation schemas from model field definitions.
package zod

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var schemaTmpl = template.Must(
	template.New("schema.ts.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/schema.ts.tmpl"),
)

var schemaTestTmpl = template.Must(
	template.New("schema.test.ts.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/schema.test.ts.tmpl"),
)

// ZodGenerator scaffolds Zod validation schemas for a resource.
type ZodGenerator struct{}

func init() {
	generators.Register(&ZodGenerator{})
}

func (g *ZodGenerator) Name() string        { return "zod" }
func (g *ZodGenerator) Description() string { return "Generate a Zod validation schema for a resource" }

func (g *ZodGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.FuncHook("install form deps", func(projectDir string) error {
			pm := "npm"
			pf, err := polafile.Load(projectDir)
			if err == nil && pf != nil && pf.PackageManager != "" {
				pm = pf.PackageManager
			}
			if pf == nil {
				pf = &polafile.Polafile{}
			}

			deps := []string{"zod"}
			args := append([]string{"install"}, deps...)

			fmt.Printf("Running: %s %s\n", pm, strings.Join(args, " "))
			cmd := exec.Command(pm, args...)
			cmd.Dir = filepath.Join(projectDir, pf.AppDir())
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}),
	}
}

func (g *ZodGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "zod [Name] [field:type ...]",
		Short: "Generate a Zod validation schema for a resource",
		Long: `Generate a TypeScript file with a Zod schema and inferred type
for the given resource. The schema is written to app/schemas/.

Field definitions follow the same syntax as the model generator.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Aliases: []string{"z"},
		Example: `  pola generate zod Product name:string price:float description:text
  pola generate zod Article title:string body:text`,
	}
}

func zodPaths(name, projectDir, appDir string, generateTests bool) []string {
	snake := schema.SnakeCase(name)
	schemasDir := filepath.Join(projectDir, appDir, "schemas")
	paths := []string{filepath.Join(schemasDir, snake+".ts")}
	if generateTests {
		paths = append(paths, filepath.Join(schemasDir, snake+".test.ts"))
	}
	return paths
}

func (g *ZodGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("name is required")
	}
	def, err := model.ParseArgs(args)
	if err != nil {
		return nil, err
	}
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		pf = &polafile.Polafile{}
	}
	genTests := generators.ShouldGenerateTests(cmd, pf.GenerateTests())
	return zodPaths(def.Name, projectDir, pf.AppDir(), genTests), nil
}

func (g *ZodGenerator) run(cmd *cobra.Command, args []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		pf = &polafile.Polafile{}
	}
	if pf.IsAPIOnly() {
		return fmt.Errorf("zod schema generation is not available in API-only mode")
	}

	def, err := model.ParseArgs(args)
	if err != nil {
		return err
	}

	data := buildZodData(def)

	schemasDir := filepath.Join(projectDir, pf.AppDir(), "schemas")
	if err := os.MkdirAll(schemasDir, 0o755); err != nil {
		return fmt.Errorf("create schemas dir: %w", err)
	}

	filePath := filepath.Join(schemasDir, data.SnakeName+".ts")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	var buf strings.Builder
	if err := schemaTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute schema template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)

	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		data.TestImport = "vitest"
		if pf.TestFramework() == "jest" {
			data.TestImport = "@jest/globals"
		}
		testPath := filepath.Join(schemasDir, data.SnakeName+".test.ts")
		if err := generators.CheckCollision(cmd, testPath); err != nil {
			return err
		}
		var testBuf strings.Builder
		if err := schemaTestTmpl.Execute(&testBuf, data); err != nil {
			return fmt.Errorf("execute schema test template: %w", err)
		}
		if err := os.WriteFile(testPath, []byte(testBuf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
		fmt.Printf("Created %s\n", testPath)
	}

	return generators.RunAfterHooks(g, projectDir)
}

// zodData holds the template data for Zod schema generation.
type zodData struct {
	Name      string     // PascalCase: "Product"
	SnakeName string     // snake_case: "product"
	Fields    []zodField // non-reference, non-bytes fields
	HasFiles  bool       // true if any field is a file upload

	// Test-template specific fields.
	TestImport        string // "vitest" or "@jest/globals"
	HasRequired       bool
	HasTypeCheck      bool
	TypeCheckField    string
	TypeCheckBadValue string
}

// zodField describes a single field for Zod schema templates.
type zodField struct {
	JSONName string // snake_case: "name"
	ZodType  string // e.g. "z.string()", "z.number().int()"
	Example  string // representative valid literal for tests, e.g. "\"hi\"", "1"
	Optional bool
	IsFile   bool // true for blob reference fields (file upload)
}

func buildZodData(def *schema.ModelDefinition) zodData {
	data := zodData{
		Name:      def.Name,
		SnakeName: schema.SnakeCase(def.Name),
		HasFiles:  def.HasBlobFields(),
	}

	for _, f := range def.Fields {
		if f.Type == schema.FieldBytes {
			continue
		}
		if f.Type == schema.FieldReferences {
			if f.RefModel == "StorageBlob" {
				data.Fields = append(data.Fields, zodField{
					JSONName: schema.SnakeCase(f.Name),
					ZodType:  "z.preprocess((v) => (v instanceof FileList ? v[0] : v), z.instanceof(File))",
					Example:  "new File([], \"a.txt\")",
					Optional: f.Optional,
					IsFile:   true,
				})
			}
			continue
		}
		zf := zodField{
			JSONName: schema.SnakeCase(f.Name),
			ZodType:  zodType(f),
			Example:  zodExample(f),
			Optional: f.Optional,
		}
		data.Fields = append(data.Fields, zf)
	}

	// Populate the test-helper fields. Find a required, non-file field whose
	// type is easy to invert for the type-mismatch test.
	for _, f := range data.Fields {
		if !f.Optional {
			data.HasRequired = true
			break
		}
	}
	for _, f := range data.Fields {
		if f.IsFile {
			continue
		}
		// Strings get a number, numbers get a string — minimal but reliable.
		if strings.Contains(f.ZodType, "string") {
			data.HasTypeCheck = true
			data.TypeCheckField = f.JSONName
			data.TypeCheckBadValue = "123"
			break
		}
		if strings.Contains(f.ZodType, "number") {
			data.HasTypeCheck = true
			data.TypeCheckField = f.JSONName
			data.TypeCheckBadValue = "\"not-a-number\""
			break
		}
		if strings.Contains(f.ZodType, "boolean") {
			data.HasTypeCheck = true
			data.TypeCheckField = f.JSONName
			data.TypeCheckBadValue = "\"yes\""
			break
		}
	}

	return data
}

// zodExample returns a representative valid literal for a field, used by the
// generated test template to construct a passing input.
func zodExample(f schema.Field) string {
	switch f.Type {
	case schema.FieldString, schema.FieldText:
		return "\"example\""
	case schema.FieldInt, schema.FieldInt64:
		return "1"
	case schema.FieldFloat:
		return "1.5"
	case schema.FieldBool:
		return "true"
	case schema.FieldTime:
		return "\"2024-01-01T00:00:00\""
	case schema.FieldUUID:
		return "\"123e4567-e89b-12d3-a456-426614174000\""
	case schema.FieldJSON:
		return "{}"
	default:
		return "undefined"
	}
}

func zodType(f schema.Field) string {
	switch f.Type {
	case schema.FieldString:
		if f.Limit > 0 {
			return fmt.Sprintf("z.string().max(%d)", f.Limit)
		}
		return "z.string()"
	case schema.FieldText:
		return "z.string()"
	case schema.FieldInt, schema.FieldInt64:
		return "z.number().int()"
	case schema.FieldFloat:
		return "z.number()"
	case schema.FieldBool:
		return "z.boolean()"
	case schema.FieldTime:
		return "z.string().datetime({ local: true })"
	case schema.FieldUUID:
		return "z.string().uuid()"
	case schema.FieldJSON:
		return "z.record(z.unknown())"
	default:
		return "z.unknown()"
	}
}

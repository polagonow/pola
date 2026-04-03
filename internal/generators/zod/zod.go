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

			deps := []string{"zod"}
			args := append([]string{"install"}, deps...)

			fmt.Printf("Running: %s %s\n", pm, strings.Join(args, " "))
			cmd := exec.Command(pm, args...)
			cmd.Dir = projectDir
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
	return generators.RunAfterHooks(g, projectDir)
}

// zodData holds the template data for Zod schema generation.
type zodData struct {
	Name      string     // PascalCase: "Product"
	SnakeName string     // snake_case: "product"
	Fields    []zodField // non-reference, non-bytes fields
}

// zodField describes a single field for Zod schema templates.
type zodField struct {
	JSONName string // snake_case: "name"
	ZodType  string // e.g. "z.string()", "z.number().int()"
	Optional bool
}

func buildZodData(def *schema.ModelDefinition) zodData {
	data := zodData{
		Name:      def.Name,
		SnakeName: schema.SnakeCase(def.Name),
	}

	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences || f.Type == schema.FieldBytes {
			continue
		}
		zf := zodField{
			JSONName: schema.SnakeCase(f.Name),
			ZodType:  zodType(f),
			Optional: f.Optional,
		}
		data.Fields = append(data.Fields, zf)
	}

	return data
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

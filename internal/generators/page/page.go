// Package page implements the page scaffold generator for the Pola CLI.
// It generates renderer-specific CRUD pages (list, show, create, edit)
// with delete support for a given resource.
package page

import (
	"embed"
	"fmt"
	"io/fs"
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

// PageGenerator scaffolds CRUD pages for a resource.
type PageGenerator struct{}

func init() {
	generators.Register(&PageGenerator{})
}

func (g *PageGenerator) Name() string                  { return "page" }
func (g *PageGenerator) Description() string           { return "Scaffold CRUD pages for a resource" }
func (g *PageGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.FuncHook("install form deps", func(projectDir string) error {
			pm := "npm"
			pf, err := polafile.Load(projectDir)
			if err == nil && pf != nil && pf.PackageManager != "" {
				pm = pf.PackageManager
			}

			deps := []string{"react-hook-form", "@hookform/resolvers"}
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

func (g *PageGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "page [Name] [field:type ...]",
		Short: "Scaffold CRUD pages for a resource",
		Long: `Generate renderer-specific CRUD pages (list, show, create, edit) with
delete support for the given resource. The renderer is read from Polafile.hcl.

Field definitions follow the same syntax as the model generator.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Aliases: []string{"p"},
		Example: `  pola generate page Product name:string price:float description:text
  pola generate page Article title:string body:text`,
	}
}

// pageSpec describes a single page file to generate.
type pageSpec struct {
	templateName string // filename inside _templates/renderers/<renderer>/
	outputPath   func(appDir, pluralSnake string) string
}

var pageSpecs = []pageSpec{
	{
		templateName: "list_page.tsx.tmpl",
		outputPath:   func(appDir, ps string) string { return filepath.Join(appDir, ps, "page.tsx") },
	},
	{
		templateName: "show_page.tsx.tmpl",
		outputPath:   func(appDir, ps string) string { return filepath.Join(appDir, ps, "[id]", "page.tsx") },
	},
	{
		templateName: "create_page.tsx.tmpl",
		outputPath:   func(appDir, ps string) string { return filepath.Join(appDir, ps, "create", "page.tsx") },
	},
	{
		templateName: "edit_page.tsx.tmpl",
		outputPath: func(appDir, ps string) string {
			return filepath.Join(appDir, ps, "[id]", "edit", "page.tsx")
		},
	},
	{
		templateName: "delete_button.tsx.tmpl",
		outputPath: func(appDir, ps string) string {
			return filepath.Join(appDir, ps, "_components", "delete-button.tsx")
		},
	},
}

func (g *PageGenerator) run(cmd *cobra.Command, args []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		return fmt.Errorf("Polafile.hcl not found; run 'pola new' to initialize a project")
	}

	renderer := pf.Renderer
	if renderer == "" {
		return fmt.Errorf("renderer not configured in Polafile.hcl; run 'pola new' to initialize a project")
	}

	// Check that we have templates for this renderer.
	rendererDir := fmt.Sprintf("_templates/renderers/%s", renderer)
	if _, err := fs.Stat(templates, rendererDir); err != nil {
		return fmt.Errorf("page templates not available for renderer %q; supported: react", renderer)
	}

	appDir := pf.AppDir()

	def, err := model.ParseArgs(args)
	if err != nil {
		return err
	}

	data := buildPageData(def)

	for _, spec := range pageSpecs {
		tmpl, err := loadTemplate(renderer, spec.templateName)
		if err != nil {
			return err
		}

		relPath := spec.outputPath(appDir, data.PluralSnake)
		absPath := filepath.Join(projectDir, relPath)

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", relPath, err)
		}

		if err := generators.CheckCollision(cmd, absPath); err != nil {
			return err
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute template %s: %w", spec.templateName, err)
		}

		if err := os.WriteFile(absPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", absPath, err)
		}

		fmt.Printf("Created %s\n", absPath)
	}

	return generators.RunAfterHooks(g, projectDir)
}

func loadTemplate(renderer, name string) (*template.Template, error) {
	path := fmt.Sprintf("_templates/renderers/%s/%s", renderer, name)
	content, err := fs.ReadFile(templates, path)
	if err != nil {
		return nil, fmt.Errorf("no %s template for renderer %q: %w", name, renderer, err)
	}
	return template.New(name).Delims("[[", "]]").Parse(string(content))
}

// pageData holds the template data for page generation.
type pageData struct {
	Name         string      // PascalCase: "Product"
	PluralName   string      // PascalCase plural: "Products"
	SnakeName    string      // snake_case: "product"
	PluralSnake  string      // snake_case plural: "products"
	SchemaImport string      // "@/schemas/product"
	Fields       []pageField // non-reference, non-bytes fields
}

// pageField describes a single field for page templates.
type pageField struct {
	Name       string // PascalCase: "Name"
	JSONName   string // snake_case (matches repo JSON tag): "name"
	Label      string // Human readable: "Name"
	InputType  string // HTML input type: "text", "number", "checkbox", etc.
	IsTextarea bool   // true for text/json fields (use <textarea> instead of <input>)
	IsFloat    bool   // true for float fields (adds step="any")
	Optional   bool
}

func buildPageData(def *schema.ModelDefinition) pageData {
	plural := schema.Pluralize(def.Name)
	snakeName := schema.SnakeCase(def.Name)
	data := pageData{
		Name:         def.Name,
		PluralName:   plural,
		SnakeName:    snakeName,
		PluralSnake:  schema.SnakeCase(plural),
		SchemaImport: "@/schemas/" + snakeName,
	}

	for _, f := range def.Fields {
		if f.Type == schema.FieldReferences || f.Type == schema.FieldBytes {
			continue
		}
		pf := pageField{
			Name:     schema.PascalCase(f.Name),
			JSONName: schema.SnakeCase(f.Name),
			Label:    humanize(f.Name),
			Optional: f.Optional,
		}
		switch f.Type {
		case schema.FieldString, schema.FieldUUID:
			pf.InputType = "text"
		case schema.FieldInt, schema.FieldInt64:
			pf.InputType = "number"
		case schema.FieldFloat:
			pf.InputType = "number"
			pf.IsFloat = true
		case schema.FieldBool:
			pf.InputType = "checkbox"
		case schema.FieldTime:
			pf.InputType = "datetime-local"
		case schema.FieldText, schema.FieldJSON:
			pf.IsTextarea = true
		}
		data.Fields = append(data.Fields, pf)
	}

	return data
}

// humanize converts a snake_case or camelCase field name to a human-readable label.
// "blog_post" → "Blog Post", "firstName" → "First Name"
func humanize(s string) string {
	// First convert to snake_case to normalize, then title-case each segment.
	snake := schema.SnakeCase(s)
	parts := strings.Split(snake, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

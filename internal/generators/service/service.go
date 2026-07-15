// Package service implements the service scaffold generator for the Pola CLI.
package service

import (
	"embed"
	"fmt"
	"os"
	"path"
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

var serviceTmpl = template.Must(
	template.New("service_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/service_go.tmpl"),
)

// ServiceGenerator scaffolds service structs with business logic methods.
type ServiceGenerator struct{}

func init() {
	generators.Register(&ServiceGenerator{})
}

func (g *ServiceGenerator) Name() string { return "service" }
func (g *ServiceGenerator) Description() string {
	return "Scaffold a service with business logic methods"
}
func (g *ServiceGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("go", "mod", "tidy"),
		generators.CmdHook("gofmt", "-w", "."),
	}
}

func (g *ServiceGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "service [Name] [field:type ...]",
		Short: "Scaffold a service with business logic methods",
		Long: `Generate a service struct that depends on a repository interface.
Field definitions follow the same syntax as the model generator.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Aliases: []string{"svc"},
		Example: `  pola generate service User name:string email:string
  pola generate service Product name:string price:float`,
	}
}

func servicePaths(name, projectDir, svcDir string) []string {
	snake := schema.SnakeCase(name)
	serviceDir := filepath.Join(projectDir, svcDir)
	return []string{filepath.Join(serviceDir, snake+"_service.go")}
}

func (g *ServiceGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("service name is required")
	}
	def, err := model.ParseArgs(args)
	if err != nil {
		return nil, err
	}
	svcDir := "services"
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load Polafile: %w", err)
	}
	if pf != nil {
		svcDir = pf.ServicesDir()
	}
	return servicePaths(def.Name, projectDir, svcDir), nil
}

func (g *ServiceGenerator) run(cmd *cobra.Command, args []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	// Read Polafile for configured service directory.
	svcDir := "services"
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf != nil {
		svcDir = pf.ServicesDir()
	}

	def, err := model.ParseArgs(args)
	if err != nil {
		return err
	}

	data := buildData(def, modulePath)
	modelsDir := "db/models"
	if pf != nil {
		modelsDir = pf.DatabaseModelsDir()
	}
	data.ModelsDir = modelsDir
	data.ModelsPkg = path.Base(modelsDir)

	serviceDir := filepath.Join(projectDir, svcDir)
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}

	filePath := filepath.Join(serviceDir, data.SnakeName+"_service.go")
	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	var buf strings.Builder
	if err := serviceTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute service template: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)

	return generators.RunAfterHooks(g, projectDir)
}

type serviceData struct {
	Name        string
	SnakeName   string
	PluralSnake string
	IDGoType    string
	ModulePath  string
	ModelsDir   string // canonical models dir (e.g. "db/models")
	ModelsPkg   string // package name of the models dir (e.g. "models")
}

func buildData(def *schema.ModelDefinition, modulePath string) serviceData {
	return serviceData{
		Name:        def.Name,
		SnakeName:   schema.SnakeCase(def.Name),
		PluralSnake: schema.SnakeCase(schema.Pluralize(def.Name)),
		IDGoType:    def.IDGoType(),
		ModulePath:  modulePath,
	}
}

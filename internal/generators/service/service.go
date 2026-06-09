// Package service implements the service scaffold generator for the Pola CLI.
package service

import (
	"embed"
	"fmt"
	"os"
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

var serviceTestTmpl = template.Must(
	template.New("service_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/service_test_go.tmpl"),
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

func servicePaths(name, projectDir, svcDir string, generateTests bool) []string {
	snake := schema.SnakeCase(name)
	serviceDir := filepath.Join(projectDir, svcDir)
	paths := []string{filepath.Join(serviceDir, snake+"_service.go")}
	if generateTests {
		paths = append(paths, filepath.Join(serviceDir, snake+"_service_test.go"))
	}
	return paths
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
	genTests := generators.ShouldGenerateTests(cmd, pf.GenerateTests())
	return servicePaths(def.Name, projectDir, svcDir, genTests), nil
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

	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		testPath := filepath.Join(serviceDir, data.SnakeName+"_service_test.go")
		if err := generators.CheckCollision(cmd, testPath); err != nil {
			return err
		}
		var testBuf strings.Builder
		if err := serviceTestTmpl.Execute(&testBuf, data); err != nil {
			return fmt.Errorf("execute service test template: %w", err)
		}
		if err := os.WriteFile(testPath, []byte(testBuf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
		fmt.Printf("Created %s\n", testPath)
	}

	return generators.RunAfterHooks(g, projectDir)
}

type serviceData struct {
	Name        string
	SnakeName   string
	PluralSnake string
	IDGoType    string
	ModulePath  string
}

func buildData(def *schema.ModelDefinition, modulePath string) serviceData {
	idGoType := "uint"
	if def.HasUUIDPrimaryKey() {
		idGoType = "string"
	}
	return serviceData{
		Name:        def.Name,
		SnakeName:   schema.SnakeCase(def.Name),
		PluralSnake: schema.SnakeCase(schema.Pluralize(def.Name)),
		IDGoType:    idGoType,
		ModulePath:  modulePath,
	}
}

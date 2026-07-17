// Package action implements the action scaffold generator for the Pola CLI.
package action

import (
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/actionbridge"
	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var (
	actionTmpl = template.Must(
		template.New("action_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/action_go.tmpl"),
	)
	actionServiceTmpl = template.Must(
		template.New("action_service_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/action_service_go.tmpl"),
	)
	actionServiceBlankTmpl = template.Must(
		template.New("action_service_blank_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/action_service_blank_go.tmpl"),
	)
	actionTestTmpl = template.Must(
		template.New("action_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/action_test_go.tmpl"),
	)
	actionServiceTestTmpl = template.Must(
		template.New("action_service_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/action_service_test_go.tmpl"),
	)
)

// ActionGenerator scaffolds new action structs in the actions/ directory.
type ActionGenerator struct{}

func init() {
	generators.Register(&ActionGenerator{})
}

func (g *ActionGenerator) Name() string        { return "action" }
func (g *ActionGenerator) Description() string { return "Scaffold a new action struct" }
func (g *ActionGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.FuncHook("regenerate bridge", func(projectDir string) error {
			pf, err := polafile.Load(projectDir)
			if err != nil || pf == nil {
				return nil
			}

			actionsDir := filepath.Join(projectDir, "actions")
			if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
				return nil
			}

			tmpDir, err := os.MkdirTemp("", "pola-bridge-*")
			if err != nil {
				return fmt.Errorf("create temp dir: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			_, err = actionbridge.Run(actionsDir, "", tmpDir, pf.PolaPackage())
			return err
		}),
		generators.CmdHook("gofmt", "-w", "."),
	}
}

func (g *ActionGenerator) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action [Name]",
		Short: "Scaffold a new action struct",
		Long: `Create a new action file in the actions/ directory with boilerplate and comments.

Use --service=Name to wire the action to a generated service.`,
		Args: cobra.ExactArgs(1),
		RunE: g.run,
		Example: `  pola generate action Blog
  pola generate action Products
  pola generate action Products --service=Product`,
	}
	cmd.Flags().String("service", "", "wire action methods to the named service")
	cmd.Flags().Bool("blank", false, "generate an empty action (no CRUD stubs); with --service the service is still injected")
	cmd.Flags().String("id-type", "uint", "Go type for the entity ID (uint or string)")
	cmd.Flags().MarkHidden("id-type")
	return cmd
}

func actionPaths(name, projectDir string, generateTests bool) []string {
	actionsDir := filepath.Join(projectDir, "actions")
	snake := schema.SnakeCase(name)
	paths := []string{filepath.Join(actionsDir, snake+"_action.go")}
	if generateTests {
		paths = append(paths, filepath.Join(actionsDir, snake+"_action_test.go"))
	}
	return paths
}

func (g *ActionGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("action name is required")
	}
	name := args[0]
	if name[0] >= 'a' && name[0] <= 'z' {
		name = string(name[0]-32) + name[1:]
	}
	pf, _ := polafile.Load(projectDir)
	genTests := generators.ShouldGenerateTests(cmd, pf.GenerateTests())
	return actionPaths(name, projectDir, genTests), nil
}

func (g *ActionGenerator) run(cmd *cobra.Command, args []string) error {
	name := args[0]
	if name[0] >= 'a' && name[0] <= 'z' {
		name = string(name[0]-32) + name[1:]
	}

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(projectDir, "actions")
	if err := os.MkdirAll(actionsDir, 0o755); err != nil {
		return fmt.Errorf("create actions dir: %w", err)
	}

	filename := schema.SnakeCase(name) + "_action.go"
	filePath := filepath.Join(actionsDir, filename)

	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	serviceName, _ := cmd.Flags().GetString("service")
	blank, _ := cmd.Flags().GetBool("blank")

	var buf strings.Builder
	switch {
	case serviceName != "":
		if serviceName[0] >= 'a' && serviceName[0] <= 'z' {
			serviceName = string(serviceName[0]-32) + serviceName[1:]
		}

		modulePath, err := project.ModulePath(projectDir)
		if err != nil {
			return fmt.Errorf("read module path: %w", err)
		}

		idType, _ := cmd.Flags().GetString("id-type")
		if idType == "" {
			idType = "uint"
		}

		if blank {
			// Service injected, but no CRUD stubs — for custom (non-entity) actions.
			if err := actionServiceBlankTmpl.Execute(&buf, struct {
				Name        string
				ServiceName string
				ModulePath  string
			}{Name: name + "Action", ServiceName: serviceName, ModulePath: modulePath}); err != nil {
				return fmt.Errorf("execute action service blank template: %w", err)
			}
		} else {
			modelsDir := "db/models"
			if pf, _ := polafile.Load(projectDir); pf != nil {
				modelsDir = pf.DatabaseModelsDir()
			}
			if err := actionServiceTmpl.Execute(&buf, struct {
				Name         string
				ServiceName  string
				IDGoType     string
				ModulePath   string
				ModelsImport string
				ModelsPkg    string
			}{
				Name:         name + "Action",
				ServiceName:  serviceName,
				IDGoType:     idType,
				ModulePath:   modulePath,
				ModelsImport: modulePath + "/" + modelsDir,
				ModelsPkg:    path.Base(modelsDir),
			}); err != nil {
				return fmt.Errorf("execute action service template: %w", err)
			}
		}
	default:
		if err := actionTmpl.Execute(&buf, struct{ Name string }{Name: name + "Action"}); err != nil {
			return fmt.Errorf("execute action template: %w", err)
		}
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)

	pf, _ := polafile.Load(projectDir)
	if !blank && generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		testFilename := schema.SnakeCase(name) + "_action_test.go"
		testPath := filepath.Join(actionsDir, testFilename)
		if err := generators.CheckCollision(cmd, testPath); err != nil {
			return err
		}
		var testBuf strings.Builder
		if serviceName != "" {
			modulePath, err := project.ModulePath(projectDir)
			if err != nil {
				return fmt.Errorf("read module path: %w", err)
			}
			idType, _ := cmd.Flags().GetString("id-type")
			if idType == "" {
				idType = "uint"
			}
			modelsDir := "db/models"
			if pf != nil {
				modelsDir = pf.DatabaseModelsDir()
			}
			if err := actionServiceTestTmpl.Execute(&testBuf, struct {
				Name        string
				ServiceName string
				IDGoType    string
				ModulePath  string
				ModelsDir   string
				ModelsPkg   string
			}{Name: name + "Action", ServiceName: serviceName, IDGoType: idType, ModulePath: modulePath, ModelsDir: modelsDir, ModelsPkg: path.Base(modelsDir)}); err != nil {
				return fmt.Errorf("execute action service test template: %w", err)
			}
		} else {
			if err := actionTestTmpl.Execute(&testBuf, struct{ Name string }{Name: name + "Action"}); err != nil {
				return fmt.Errorf("execute action test template: %w", err)
			}
		}
		if err := os.WriteFile(testPath, []byte(testBuf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
		fmt.Printf("Created %s\n", testPath)
	}

	return generators.RunAfterHooks(g, projectDir)
}

// Package action implements the action scaffold generator for the Pola CLI.
package action

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/actionbridge"
	"github.com/polagonow/pola/internal/generators"
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
)

// ActionGenerator scaffolds new action structs in the actions/ directory.
type ActionGenerator struct{}

func init() {
	generators.Register(&ActionGenerator{})
}

func (g *ActionGenerator) Name() string        { return "action" }
func (g *ActionGenerator) Description() string  { return "Scaffold a new action struct" }
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
	return cmd
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

	filename := strings.ToLower(name) + ".go"
	filePath := filepath.Join(actionsDir, filename)

	if err := generators.CheckCollision(cmd, filePath); err != nil {
		return err
	}

	serviceName, _ := cmd.Flags().GetString("service")

	var buf strings.Builder
	if serviceName != "" {
		if serviceName[0] >= 'a' && serviceName[0] <= 'z' {
			serviceName = string(serviceName[0]-32) + serviceName[1:]
		}

		modulePath, err := project.ModulePath(projectDir)
		if err != nil {
			return fmt.Errorf("read module path: %w", err)
		}

		if err := actionServiceTmpl.Execute(&buf, struct {
			Name        string
			ServiceName string
			ModulePath  string
		}{
			Name:        name,
			ServiceName: serviceName,
			ModulePath:  modulePath,
		}); err != nil {
			return fmt.Errorf("execute action service template: %w", err)
		}
	} else {
		if err := actionTmpl.Execute(&buf, struct{ Name string }{Name: name}); err != nil {
			return fmt.Errorf("execute action template: %w", err)
		}
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)
	return generators.RunAfterHooks(g, projectDir)
}

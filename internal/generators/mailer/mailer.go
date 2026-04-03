// Package mailer implements the mailer scaffold generator for the Pola CLI.
package mailer

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/generators/model/schema"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

var (
	mailerGoTmpl = template.Must(
		template.New("mailer_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_go.tmpl"),
	)
	templateTsxTmpl = template.Must(
		template.New("mailer_template_tsx.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_template_tsx.tmpl"),
	)
	layoutTsxTmpl = template.Must(
		template.New("mailer_layout_tsx.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_layout_tsx.tmpl"),
	)
)

// MailerGenerator scaffolds mailer structs and react.email templates.
type MailerGenerator struct{}

func init() {
	generators.Register(&MailerGenerator{})
}

func (g *MailerGenerator) Name() string        { return "mailer" }
func (g *MailerGenerator) Description() string { return "Scaffold a mailer with email templates" }
func (g *MailerGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("go", "mod", "tidy"),
		generators.FuncHook("install react-email packages", installReactEmail),
	}
}

func (g *MailerGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "mailer [Name] [actions...]",
		Short: "Scaffold a mailer with email templates",
		Long: `Generate a mailer Go struct and React email templates.

Each action argument becomes a method on the mailer and a corresponding
.tsx template under app/mailers/<name>_mailer/.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Example: `  pola generate mailer User welcome reset_password
  pola generate mailer Order confirmation shipped`,
	}
}

func (g *MailerGenerator) run(cmd *cobra.Command, args []string) error {
	name := schema.PascalCase(args[0])
	// Strip "Mailer" suffix if provided — we add it in the template.
	name = strings.TrimSuffix(name, "Mailer")

	actions := args[1:]
	if len(actions) == 0 {
		return fmt.Errorf("at least one action name is required (e.g. welcome, reset_password)")
	}

	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	modulePath, err := project.ModulePath(projectDir)
	if err != nil {
		return fmt.Errorf("read module path: %w", err)
	}

	// Determine app dir from Polafile.
	appDir := "app"
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf != nil && pf.App != "" {
		appDir = pf.App
	}

	snakeName := schema.SnakeCase(name)

	// Build action data.
	actionData := make([]actionInfo, len(actions))
	for i, a := range actions {
		actionData[i] = actionInfo{
			PascalName: schema.PascalCase(a),
			SnakeName:  schema.SnakeCase(schema.PascalCase(a)),
			HumanName:  humanize(a),
		}
	}

	data := mailerData{
		Name:       name,
		SnakeName:  snakeName,
		ModulePath: modulePath,
		Actions:    actionData,
	}

	// 1. Generate Go mailer file.
	mailersDir := filepath.Join(projectDir, "mailers")
	if err := os.MkdirAll(mailersDir, 0o755); err != nil {
		return fmt.Errorf("create mailers dir: %w", err)
	}

	goFilePath := filepath.Join(mailersDir, snakeName+"_mailer.go")
	if err := generators.CheckCollision(cmd, goFilePath); err != nil {
		return err
	}

	var buf strings.Builder
	if err := mailerGoTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute mailer template: %w", err)
	}
	if err := os.WriteFile(goFilePath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", goFilePath, err)
	}
	fmt.Printf("Created %s\n", goFilePath)

	// 2. Generate TSX templates for each action.
	tsxDir := filepath.Join(projectDir, appDir, "mailers", snakeName+"_mailer")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}

	for _, action := range actionData {
		tsxPath := filepath.Join(tsxDir, action.SnakeName+".tsx")
		if err := generators.CheckCollision(cmd, tsxPath); err != nil {
			return err
		}

		buf.Reset()
		if err := templateTsxTmpl.Execute(&buf, action); err != nil {
			return fmt.Errorf("execute template tsx: %w", err)
		}
		if err := os.WriteFile(tsxPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", tsxPath, err)
		}
		fmt.Printf("Created %s\n", tsxPath)
	}

	// 3. Generate default layout if it doesn't exist.
	layoutDir := filepath.Join(projectDir, appDir, "mailers", "layouts")
	layoutPath := filepath.Join(layoutDir, "default.tsx")
	if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
		if err := os.MkdirAll(layoutDir, 0o755); err != nil {
			return fmt.Errorf("create layouts dir: %w", err)
		}
		buf.Reset()
		if err := layoutTsxTmpl.Execute(&buf, nil); err != nil {
			return fmt.Errorf("execute layout template: %w", err)
		}
		if err := os.WriteFile(layoutPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", layoutPath, err)
		}
		fmt.Printf("Created %s\n", layoutPath)
	}

	return generators.RunAfterHooks(g, projectDir)
}

type mailerData struct {
	Name       string
	SnakeName  string
	ModulePath string
	Actions    []actionInfo
}

type actionInfo struct {
	PascalName string
	SnakeName  string
	HumanName  string
}

// humanize converts "reset_password" → "Reset Password".
func humanize(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = string(unicode.ToUpper(rune(w[0]))) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// installReactEmail installs @react-email/components and @react-email/render
// using the project's configured package manager.
func installReactEmail(projectDir string) error {
	pm := "npm"
	pf, err := polafile.Load(projectDir)
	if err == nil && pf != nil && pf.PackageManager != "" {
		pm = pf.PackageManager
	}

	var args []string
	switch pm {
	case "pnpm":
		args = []string{"pnpm", "add", "@react-email/components", "@react-email/render", "react"}
	case "yarn":
		args = []string{"yarn", "add", "@react-email/components", "@react-email/render", "react"}
	case "bun":
		args = []string{"bun", "add", "@react-email/components", "@react-email/render", "react"}
	default:
		args = []string{"npm", "install", "@react-email/components", "@react-email/render", "react"}
	}

	return generators.RunAfterHooks(&npmRunner{args: args}, projectDir)
}

// npmRunner is a minimal Generator implementation used to run npm install
// as an after-hook.
type npmRunner struct {
	args []string
}

func (r *npmRunner) Name() string                      { return "npm-install" }
func (r *npmRunner) Description() string               { return "Install npm packages" }
func (r *npmRunner) Command() *cobra.Command            { return nil }
func (r *npmRunner) AfterHooks() []generators.Hook {
	return []generators.Hook{generators.CmdHook(r.args...)}
}

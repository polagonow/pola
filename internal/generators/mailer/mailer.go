// Package mailer implements the mailer scaffold generator for the Pola CLI.
package mailer

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
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
	mailerTestTmpl = template.Must(
		template.New("mailer_test_go.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_test_go.tmpl"),
	)
	// React email templates.
	templateTsxTmpl = template.Must(
		template.New("mailer_template_tsx.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_template_tsx.tmpl"),
	)
	layoutTsxTmpl = template.Must(
		template.New("mailer_layout_tsx.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_layout_tsx.tmpl"),
	)
	// Go template scaffolds.
	templateHTMLTmplTmpl = template.Must(
		template.New("mailer_template_html_tmpl.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_template_html_tmpl.tmpl"),
	)
	templateTextTmplTmpl = template.Must(
		template.New("mailer_template_text_tmpl.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_template_text_tmpl.tmpl"),
	)
	layoutHTMLTmplTmpl = template.Must(
		template.New("mailer_layout_html_tmpl.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_layout_html_tmpl.tmpl"),
	)
	layoutTextTmplTmpl = template.Must(
		template.New("mailer_layout_text_tmpl.tmpl").Delims("[[", "]]").ParseFS(templates, "_templates/mailer_layout_text_tmpl.tmpl"),
	)
)

// MailerGenerator scaffolds mailer structs and email templates.
type MailerGenerator struct{}

func init() {
	generators.Register(&MailerGenerator{})
}

func (g *MailerGenerator) Name() string        { return "mailer" }
func (g *MailerGenerator) Description() string { return "Scaffold a mailer with email templates" }
func (g *MailerGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.CmdHook("go", "mod", "tidy"),
	}
}

func (g *MailerGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "mailer [Name] [actions...]",
		Short: "Scaffold a mailer with email templates",
		Long: `Generate a mailer Go struct and email templates.

Each action argument becomes a method on the mailer and a corresponding
template under app/mailers/<name>_mailer/.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    g.run,
		Example: `  pola generate mailer User welcome reset_password
  pola generate mailer Order confirmation shipped`,
	}
}

func (g *MailerGenerator) Artifacts(cmd *cobra.Command, args []string, projectDir string) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("mailer name is required")
	}
	name := schema.PascalCase(args[0])
	name = strings.TrimSuffix(name, "Mailer")
	actions := args[1:]
	if len(actions) == 0 {
		return nil, fmt.Errorf("at least one action name is required")
	}

	appDir := "app"
	rendererType := "react"
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load Polafile: %w", err)
	}
	if pf != nil {
		if pf.App != "" {
			appDir = pf.App
		}
		rendererType = pf.MailerRenderer("development")
	}

	snakeName := schema.SnakeCase(name)
	var paths []string

	paths = append(paths, filepath.Join(projectDir, "mailers", snakeName+"_mailer.go"))
	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		paths = append(paths, filepath.Join(projectDir, "mailers", snakeName+"_mailer_test.go"))
	}

	tmplDir := filepath.Join(projectDir, appDir, "mailers", snakeName+"_mailer")
	for _, a := range actions {
		actionSnake := schema.SnakeCase(schema.PascalCase(a))
		if rendererType == "tmpl" {
			paths = append(paths,
				filepath.Join(tmplDir, actionSnake+".html.tmpl"),
				filepath.Join(tmplDir, actionSnake+".text.tmpl"),
			)
		} else {
			paths = append(paths, filepath.Join(tmplDir, actionSnake+".tsx"))
		}
	}

	return paths, nil
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

	// Determine app dir and renderer from Polafile.
	appDir := "app"
	rendererType := "react"
	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf != nil {
		if pf.App != "" {
			appDir = pf.App
		}
		rendererType = pf.MailerRenderer("development")
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

	polaPackage := polafile.DefaultPackage
	if pf != nil {
		polaPackage = pf.PolaPackage()
	}

	data := mailerData{
		Name:        name,
		SnakeName:   snakeName,
		ModulePath:  modulePath,
		PolaPackage: polaPackage,
		Actions:     actionData,
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

	// Ensure a mailer {} block exists so the framework wires the renderer +
	// transport (and provides mailer.Base). Without it the generated mailer is a
	// dead scaffold that fails DI. Default to the "log" transport for dev. Done
	// here (right after the Go file) so it runs even if later template steps fail.
	if mpf, _ := polafile.Load(projectDir); mpf != nil && mpf.Mailer == nil {
		mpf.Mailer = &polafile.Mailer{Renderer: rendererType, Transport: "log", From: "noreply@example.com"}
		if err := polafile.Save(projectDir, mpf); err != nil {
			return fmt.Errorf("add mailer block to Polafile: %w", err)
		}
		fmt.Printf("Added mailer {} block to Polafile.hcl (transport=log, renderer=%s).\n", rendererType)
	}

	if generators.ShouldGenerateTests(cmd, pf.GenerateTests()) {
		testPath := filepath.Join(mailersDir, snakeName+"_mailer_test.go")
		if err := generators.CheckCollision(cmd, testPath); err != nil {
			return err
		}
		buf.Reset()
		if err := mailerTestTmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute mailer test template: %w", err)
		}
		if err := os.WriteFile(testPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", testPath, err)
		}
		fmt.Printf("Created %s\n", testPath)
	}

	// 2. Generate email templates for each action.
	tmplDir := filepath.Join(projectDir, appDir, "mailers", snakeName+"_mailer")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}

	if rendererType == "tmpl" {
		if err := scaffoldTmplTemplates(cmd, tmplDir, actionData, &buf); err != nil {
			return err
		}
		if err := scaffoldTmplLayouts(cmd, projectDir, appDir, &buf); err != nil {
			return err
		}
	} else {
		if err := scaffoldReactTemplates(cmd, tmplDir, actionData, &buf); err != nil {
			return err
		}
		if err := scaffoldReactLayout(cmd, projectDir, appDir, &buf); err != nil {
			return err
		}
		// Install react-email packages.
		if err := installReactEmail(projectDir); err != nil {
			return err
		}
	}

	return generators.RunAfterHooks(g, projectDir)
}

func scaffoldReactTemplates(cmd *cobra.Command, dir string, actions []actionInfo, buf *strings.Builder) error {
	for _, action := range actions {
		tsxPath := filepath.Join(dir, action.SnakeName+".tsx")
		if err := generators.CheckCollision(cmd, tsxPath); err != nil {
			return err
		}

		buf.Reset()
		if err := templateTsxTmpl.Execute(buf, action); err != nil {
			return fmt.Errorf("execute template tsx: %w", err)
		}
		if err := os.WriteFile(tsxPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", tsxPath, err)
		}
		fmt.Printf("Created %s\n", tsxPath)
	}
	return nil
}

func scaffoldReactLayout(cmd *cobra.Command, projectDir, appDir string, buf *strings.Builder) error {
	layoutDir := filepath.Join(projectDir, appDir, "mailers", "layouts")
	layoutPath := filepath.Join(layoutDir, "default.tsx")
	if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
		if err := os.MkdirAll(layoutDir, 0o755); err != nil {
			return fmt.Errorf("create layouts dir: %w", err)
		}
		buf.Reset()
		if err := layoutTsxTmpl.Execute(buf, nil); err != nil {
			return fmt.Errorf("execute layout template: %w", err)
		}
		if err := os.WriteFile(layoutPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", layoutPath, err)
		}
		fmt.Printf("Created %s\n", layoutPath)
	}
	return nil
}

func scaffoldTmplTemplates(cmd *cobra.Command, dir string, actions []actionInfo, buf *strings.Builder) error {
	for _, action := range actions {
		// HTML template.
		htmlPath := filepath.Join(dir, action.SnakeName+".html.tmpl")
		if err := generators.CheckCollision(cmd, htmlPath); err != nil {
			return err
		}

		buf.Reset()
		if err := templateHTMLTmplTmpl.Execute(buf, action); err != nil {
			return fmt.Errorf("execute html tmpl: %w", err)
		}
		if err := os.WriteFile(htmlPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", htmlPath, err)
		}
		fmt.Printf("Created %s\n", htmlPath)

		// Text template.
		textPath := filepath.Join(dir, action.SnakeName+".text.tmpl")
		if err := generators.CheckCollision(cmd, textPath); err != nil {
			return err
		}

		buf.Reset()
		if err := templateTextTmplTmpl.Execute(buf, action); err != nil {
			return fmt.Errorf("execute text tmpl: %w", err)
		}
		if err := os.WriteFile(textPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", textPath, err)
		}
		fmt.Printf("Created %s\n", textPath)
	}
	return nil
}

func scaffoldTmplLayouts(cmd *cobra.Command, projectDir, appDir string, buf *strings.Builder) error {
	layoutDir := filepath.Join(projectDir, appDir, "mailers", "layouts")

	// HTML layout.
	htmlLayoutPath := filepath.Join(layoutDir, "default.html.tmpl")
	if _, err := os.Stat(htmlLayoutPath); os.IsNotExist(err) {
		if err := os.MkdirAll(layoutDir, 0o755); err != nil {
			return fmt.Errorf("create layouts dir: %w", err)
		}
		buf.Reset()
		if err := layoutHTMLTmplTmpl.Execute(buf, nil); err != nil {
			return fmt.Errorf("execute html layout tmpl: %w", err)
		}
		if err := os.WriteFile(htmlLayoutPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", htmlLayoutPath, err)
		}
		fmt.Printf("Created %s\n", htmlLayoutPath)
	}

	// Text layout.
	textLayoutPath := filepath.Join(layoutDir, "default.text.tmpl")
	if _, err := os.Stat(textLayoutPath); os.IsNotExist(err) {
		buf.Reset()
		if err := layoutTextTmplTmpl.Execute(buf, nil); err != nil {
			return fmt.Errorf("execute text layout tmpl: %w", err)
		}
		if err := os.WriteFile(textLayoutPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", textLayoutPath, err)
		}
		fmt.Printf("Created %s\n", textLayoutPath)
	}

	return nil
}

type mailerData struct {
	Name        string
	SnakeName   string
	ModulePath  string
	PolaPackage string
	Actions     []actionInfo
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
// using the project's configured package manager, in the web/ directory.
func installReactEmail(projectDir string) error {
	pm := "npm"
	pf, err := polafile.Load(projectDir)
	if err == nil && pf != nil && pf.PackageManager != "" {
		pm = pf.PackageManager
	}
	if pf == nil {
		pf = &polafile.Polafile{}
	}

	deps := []string{"@react-email/components", "@react-email/render", "react"}
	var args []string
	switch pm {
	case "pnpm", "yarn", "bun":
		args = append([]string{"add"}, deps...)
	default:
		args = append([]string{"install"}, deps...)
	}

	fmt.Printf("Running: %s %s\n", pm, strings.Join(args, " "))
	cmd := exec.Command(pm, args...)
	cmd.Dir = filepath.Join(projectDir, pf.AppDir())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

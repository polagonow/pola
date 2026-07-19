// Package docs implements the `pola generate docs` scaffold generator: it adds a
// Fumadocs documentation section to an existing Pola app. Content is authored as
// Markdown/MDX under content/docs and compiled by Pola's Go-native pipeline (no
// Node.js); the /docs routes use the real fumadocs-core + fumadocs-ui packages.
package docs

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

//go:embed all:_templates
var templates embed.FS

// DocsGenerator scaffolds a Fumadocs documentation section.
type DocsGenerator struct{}

func init() { generators.Register(&DocsGenerator{}) }

func (g *DocsGenerator) Name() string        { return "docs" }
func (g *DocsGenerator) Description() string { return "Scaffold a Fumadocs documentation section" }

func (g *DocsGenerator) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "docs",
		Short: "Scaffold a Fumadocs documentation section (content/docs + /docs routes)",
		Long: `Generate a Fumadocs-powered documentation section:

  - content/docs/*.mdx      Markdown pages (compiled by Pola's Go MDX pipeline)
  - content/docs/meta.json  sidebar order
  - lib/source.ts           content source wired to fumadocs-core's loader()
  - app/docs/layout.tsx     fumadocs-ui DocsLayout (sidebar/nav) + providers
  - app/docs/[[...slug]]/page.tsx

MDX is compiled by Pola in Go — no fumadocs-mdx / Node.js. The real fumadocs-core
and fumadocs-ui packages are installed and rendered through React Server Components.`,
		Args:    cobra.NoArgs,
		RunE:    g.run,
		Example: `  pola generate docs`,
	}
}

func (g *DocsGenerator) AfterHooks() []generators.Hook {
	return []generators.Hook{
		generators.FuncHook("install fumadocs deps", func(projectDir string) error {
			pm, appDir := "npm", "web"
			if pf, err := polafile.Load(projectDir); err == nil && pf != nil {
				if pf.PackageManager != "" {
					pm = pf.PackageManager
				}
				appDir = pf.AppDir()
			}
			args := []string{"install", "fumadocs-core@16", "fumadocs-ui@16", "lucide-react"}
			fmt.Printf("Running: %s %s\n", pm, strings.Join(args, " "))
			cmd := exec.Command(pm, args...)
			cmd.Dir = filepath.Join(projectDir, appDir)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			return cmd.Run()
		}),
	}
}

// docsFile is one scaffolded file: an embedded template and its output path
// relative to the web app directory.
type docsFile struct {
	tmpl string
	out  func(appDir string) string
}

var docsFiles = []docsFile{
	{"index.mdx.tmpl", func(a string) string { return filepath.Join(a, "content", "docs", "index.mdx") }},
	{"getting-started.mdx.tmpl", func(a string) string { return filepath.Join(a, "content", "docs", "getting-started.mdx") }},
	{"meta.json.tmpl", func(a string) string { return filepath.Join(a, "content", "docs", "meta.json") }},
	{"source.ts.tmpl", func(a string) string { return filepath.Join(a, "lib", "source.ts") }},
	{"layout.tsx.tmpl", func(a string) string { return filepath.Join(a, "app", "docs", "layout.tsx") }},
	{"page.tsx.tmpl", func(a string) string { return filepath.Join(a, "app", "docs", "[[...slug]]", "page.tsx") }},
}

func (g *DocsGenerator) run(cmd *cobra.Command, _ []string) error {
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
	if pf.IsAPIOnly() {
		return fmt.Errorf("docs generation is not available in API-only mode")
	}
	if pf.Renderer == "" || !strings.HasPrefix(pf.Renderer, "react") {
		return fmt.Errorf("docs generation requires the react renderer (found %q)", pf.Renderer)
	}
	appDir := pf.AppDir()

	for _, f := range docsFiles {
		tmpl, terr := template.New(f.tmpl).Delims("[[", "]]").ParseFS(templates, "_templates/"+f.tmpl)
		if terr != nil {
			return fmt.Errorf("parse template %s: %w", f.tmpl, terr)
		}
		abs := filepath.Join(projectDir, f.out(appDir))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.tmpl, err)
		}
		if err := generators.CheckCollision(cmd, abs); err != nil {
			return err
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, nil); err != nil {
			return fmt.Errorf("execute template %s: %w", f.tmpl, err)
		}
		if err := os.WriteFile(abs, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", abs, err)
		}
		fmt.Printf("Created %s\n", abs)
	}

	// Add fumadocs-ui's Tailwind preset to the app stylesheet so its classes are
	// generated. Best-effort: on failure, print the lines to add by hand.
	cssPath := filepath.Join(projectDir, appDir, "app", "globals.css")
	if err := ensureFumadocsCSS(cssPath); err != nil {
		fmt.Printf("note: update %s manually — add after '@import \"tailwindcss\";':\n", cssPath)
		fmt.Println(`  @import "fumadocs-ui/css/neutral.css";`)
		fmt.Println(`  @import "fumadocs-ui/css/preset.css";`)
	} else {
		fmt.Printf("Updated %s\n", cssPath)
	}

	fmt.Println("\nDocs section ready. Run `pola dev` and open /docs.")
	return generators.RunAfterHooks(g, projectDir)
}

// ensureFumadocsCSS inserts the fumadocs-ui theme + preset imports right after
// the Tailwind import in globals.css. It is idempotent and leaves the file
// untouched if the preset is already imported.
func ensureFumadocsCSS(cssPath string) error {
	data, err := os.ReadFile(cssPath)
	if err != nil {
		return err
	}
	s := string(data)
	if strings.Contains(s, "fumadocs-ui/css/preset.css") {
		return nil
	}
	const marker = `@import "tailwindcss";`
	if !strings.Contains(s, marker) {
		return fmt.Errorf("no %q found in globals.css", marker)
	}
	replacement := marker + "\n" +
		`@import "fumadocs-ui/css/neutral.css";` + "\n" +
		`@import "fumadocs-ui/css/preset.css";`
	return os.WriteFile(cssPath, []byte(strings.Replace(s, marker, replacement, 1)), 0o644)
}

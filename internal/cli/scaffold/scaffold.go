// Package scaffold provides project scaffolding for the pola CLI.
package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:_templates
var templates embed.FS

// Data holds the template variables for scaffolding a new project.
type Data struct {
	AppName       string
	ModulePath    string
	PolaPackage   string // e.g. "github.com/polagonow/pola"
	Renderer      string
	Bundler       string
	Router        string
	CSS           string
	VM            string
	PolaVersion   string
	PolaLocalPath string // if set, generates a replace directive in go.mod
}

// Execute renders all embedded templates into targetDir.
// It first copies shared templates (everything outside renderers/),
// then overlays renderer-specific templates from renderers/<renderer>/.
func Execute(targetDir string, data Data) error {
	// Pass 1: shared templates (skip the renderers/ subtree).
	if err := renderTree(templates, "_templates", targetDir, data, func(rel string) (string, bool) {
		if rel == "renderers" || strings.HasPrefix(rel, "renderers/") || strings.HasPrefix(rel, "renderers\\") {
			return "", false // skip
		}
		return rel, true
	}); err != nil {
		return fmt.Errorf("shared templates: %w", err)
	}

	// Pass 2: renderer-specific templates.
	rendererRoot := filepath.Join("_templates", "renderers", data.Renderer)
	if _, err := fs.Stat(templates, rendererRoot); err != nil {
		return fmt.Errorf("no templates for renderer %q", data.Renderer)
	}
	if err := renderTree(templates, rendererRoot, targetDir, data, func(rel string) (string, bool) {
		return rel, true
	}); err != nil {
		return fmt.Errorf("renderer %s templates: %w", data.Renderer, err)
	}

	return nil
}

// pathFilter returns the output relative path and whether to include the file.
type pathFilter func(rel string) (outRel string, include bool)

// renderTree walks root in the embedded FS and writes rendered files to targetDir.
func renderTree(fsys embed.FS, root, targetDir string, data Data, filter pathFilter) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		outRel, include := filter(rel)
		if !include {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		outPath := filepath.Join(targetDir, outRel)

		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}

		// Strip .tmpl extension for output filename.
		outPath = strings.TrimSuffix(outPath, ".tmpl")

		// Restore underscored Go filenames: main_go -> main.go, embed_go -> embed.go
		// (We use _go.tmpl instead of .go.tmpl to prevent the Go toolchain from
		// treating embedded templates as Go source files.)
		base := filepath.Base(outPath)
		if strings.HasSuffix(base, "_go") {
			restored := strings.TrimSuffix(base, "_go") + ".go"
			outPath = filepath.Join(filepath.Dir(outPath), restored)
		}

		// Handle special filenames that can't use their real names in the
		// embedded FS (they'd confuse the Go toolchain).
		switch filepath.Base(outPath) {
		case "gitignore":
			outPath = filepath.Join(filepath.Dir(outPath), ".gitignore")
		case "gomod":
			outPath = filepath.Join(filepath.Dir(outPath), "go.mod")
		}

		// Read template source.
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		// If it's a .tmpl file, execute as template. Otherwise copy as-is.
		var output []byte
		if strings.HasSuffix(path, ".tmpl") {
			tmpl, err := template.New(filepath.Base(path)).Delims("[[", "]]").Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse template %s: %w", path, err)
			}
			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err != nil {
				return fmt.Errorf("execute template %s: %w", path, err)
			}
			output = []byte(buf.String())
		} else {
			output = content
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}

		return os.WriteFile(outPath, output, 0o644)
	})
}

package mailer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EmailTemplate describes a discovered email template component.
type EmailTemplate struct {
	Name    string // e.g. "user_mailer/welcome"
	AbsPath string // absolute file path
}

// EmailLayout describes a discovered email layout component.
type EmailLayout struct {
	Name    string // e.g. "default"
	AbsPath string // absolute file path
}

// ScanMailers discovers email templates and layouts under the given mailers
// directory. Templates are organized as mailers/<mailer_name>/<action>.tsx
// and layouts live under mailers/layouts/.
func ScanMailers(mailersDir string, exts []string) ([]EmailTemplate, []EmailLayout, error) {
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		extSet[e] = true
	}

	var templates []EmailTemplate
	var layouts []EmailLayout

	entries, err := os.ReadDir(mailersDir)
	if err != nil {
		return nil, nil, fmt.Errorf("mailer: read mailers dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		subDir := filepath.Join(mailersDir, dirName)

		files, err := os.ReadDir(subDir)
		if err != nil {
			return nil, nil, fmt.Errorf("mailer: read %s: %w", subDir, err)
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ext := filepath.Ext(f.Name())
			if !extSet[ext] {
				continue
			}
			baseName := strings.TrimSuffix(f.Name(), ext)
			absPath := filepath.Join(subDir, f.Name())

			if dirName == "layouts" {
				layouts = append(layouts, EmailLayout{
					Name:    baseName,
					AbsPath: absPath,
				})
			} else {
				templates = append(templates, EmailTemplate{
					Name:    dirName + "/" + baseName,
					AbsPath: absPath,
				})
			}
		}
	}

	return templates, layouts, nil
}

// GenerateEmailEntry produces the TypeScript entry source for the email bundle.
// This entry imports all templates and layouts and exposes a __renderEmail__
// global that renders a template with optional layout wrapping via @react-email/render.
func GenerateEmailEntry(templates []EmailTemplate, layouts []EmailLayout, mailersDir string) string {
	var b strings.Builder

	b.WriteString("import React from 'react';\n")
	b.WriteString("import { render } from '@react-email/render';\n\n")

	// Import templates.
	for i, t := range templates {
		rel := relImportPath(mailersDir, t.AbsPath)
		b.WriteString(fmt.Sprintf("import Template%d from '%s';\n", i, rel))
	}

	// Import layouts.
	for i, l := range layouts {
		rel := relImportPath(mailersDir, l.AbsPath)
		b.WriteString(fmt.Sprintf("import Layout%d from '%s';\n", i, rel))
	}

	// Template map.
	b.WriteString("\nconst templates: Record<string, React.ComponentType<any>> = {\n")
	for i, t := range templates {
		b.WriteString(fmt.Sprintf("  '%s': Template%d,\n", t.Name, i))
	}
	b.WriteString("};\n\n")

	// Layout map.
	b.WriteString("const layouts: Record<string, React.ComponentType<any>> = {\n")
	for i, l := range layouts {
		b.WriteString(fmt.Sprintf("  '%s': Layout%d,\n", l.Name, i))
	}
	b.WriteString("};\n\n")

	// __renderEmail__ global function.
	b.WriteString(`(globalThis as any).__renderEmail__ = function(
  templateName: string,
  layoutName: string,
  propsJSON: string
): { html: string; text: string } {
  const Template = templates[templateName];
  if (!Template) {
    throw new Error('Unknown email template: ' + templateName);
  }

  const props = propsJSON ? JSON.parse(propsJSON) : {};
  let element = React.createElement(Template, props);

  if (layoutName && layouts[layoutName]) {
    const Layout = layouts[layoutName];
    element = React.createElement(Layout, {}, element);
  }

  const html = render(element);
  const text = render(element, { plainText: true });
  return { html, text };
};
`)

	return b.String()
}

// relImportPath computes a relative import path from the mailers directory
// to the given absolute file path, stripping the file extension.
func relImportPath(mailersDir, absPath string) string {
	rel, err := filepath.Rel(filepath.Dir(mailersDir), absPath)
	if err != nil {
		return absPath
	}
	// Strip extension for import.
	ext := filepath.Ext(rel)
	rel = strings.TrimSuffix(rel, ext)
	// Ensure forward slashes and leading ./
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

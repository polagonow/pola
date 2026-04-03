package tmpl

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/mailer"
)

// Renderer renders email templates using Go's html/template and text/template.
type Renderer struct {
	logger core.Logger

	mu          sync.RWMutex
	mailersDir  string
	htmlTmpls   map[string]*htmltemplate.Template // keyed by e.g. "user_mailer/welcome"
	textTmpls   map[string]*texttemplate.Template
	htmlLayouts map[string]*htmltemplate.Template // keyed by e.g. "default"
	textLayouts map[string]*texttemplate.Template
}

// Plugin returns a core.Plugin that registers the Go template renderer
// as the mailer.EmailRenderer and mailer.TemplateLoader in the DI container.
func Plugin() core.Plugin {
	return core.PluginFunc{
		PluginName: "mailer.renderer.tmpl",
		Fn: func(r *core.Registry) {
			logger, _ := core.Invoke[core.Logger](r)
			renderer := &Renderer{logger: logger}
			core.ProvideValue[mailer.EmailRenderer](r, renderer)
			core.ProvideValue[mailer.TemplateLoader](r, renderer)
			core.ProvideValue[*Renderer](r, renderer)
		},
	}
}

// LoadTemplates scans the mailers directory for .html.tmpl and .text.tmpl
// files and parses them into cached template sets.
func (r *Renderer) LoadTemplates(mailersDir string) error {
	htmlTmpls := make(map[string]*htmltemplate.Template)
	textTmpls := make(map[string]*texttemplate.Template)
	htmlLayouts := make(map[string]*htmltemplate.Template)
	textLayouts := make(map[string]*texttemplate.Template)

	entries, err := os.ReadDir(mailersDir)
	if err != nil {
		return fmt.Errorf("tmpl: read mailers dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		subDir := filepath.Join(mailersDir, dirName)

		files, err := os.ReadDir(subDir)
		if err != nil {
			return fmt.Errorf("tmpl: read %s: %w", subDir, err)
		}

		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".tmpl" {
				continue
			}

			absPath := filepath.Join(subDir, f.Name())
			name := strings.TrimSuffix(f.Name(), ".tmpl")

			if dirName == "layouts" {
				if err := parseLayout(name, absPath, htmlLayouts, textLayouts); err != nil {
					return err
				}
			} else {
				if err := parseTemplate(dirName, name, absPath, htmlTmpls, textTmpls); err != nil {
					return err
				}
			}
		}
	}

	r.mu.Lock()
	r.mailersDir = mailersDir
	r.htmlTmpls = htmlTmpls
	r.textTmpls = textTmpls
	r.htmlLayouts = htmlLayouts
	r.textLayouts = textLayouts
	r.mu.Unlock()

	total := len(htmlTmpls) + len(textTmpls)
	r.logger.Info("pola: email templates loaded", "templates", total, "html_layouts", len(htmlLayouts), "text_layouts", len(textLayouts))
	return nil
}

func parseLayout(name, absPath string, htmlLayouts map[string]*htmltemplate.Template, textLayouts map[string]*texttemplate.Template) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("tmpl: read layout %s: %w", absPath, err)
	}
	src := string(data)

	if strings.HasSuffix(name, ".html") {
		key := strings.TrimSuffix(name, ".html")
		t, err := htmltemplate.New("layout").Parse(src)
		if err != nil {
			return fmt.Errorf("tmpl: parse html layout %s: %w", absPath, err)
		}
		htmlLayouts[key] = t
	} else if strings.HasSuffix(name, ".text") {
		key := strings.TrimSuffix(name, ".text")
		t, err := texttemplate.New("layout").Parse(src)
		if err != nil {
			return fmt.Errorf("tmpl: parse text layout %s: %w", absPath, err)
		}
		textLayouts[key] = t
	}
	return nil
}

func parseTemplate(dirName, name, absPath string, htmlTmpls map[string]*htmltemplate.Template, textTmpls map[string]*texttemplate.Template) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("tmpl: read template %s: %w", absPath, err)
	}
	src := string(data)

	if strings.HasSuffix(name, ".html") {
		key := dirName + "/" + strings.TrimSuffix(name, ".html")
		t, err := htmltemplate.New("content").Parse(src)
		if err != nil {
			return fmt.Errorf("tmpl: parse html template %s: %w", absPath, err)
		}
		htmlTmpls[key] = t
	} else if strings.HasSuffix(name, ".text") {
		key := dirName + "/" + strings.TrimSuffix(name, ".text")
		t, err := texttemplate.New("content").Parse(src)
		if err != nil {
			return fmt.Errorf("tmpl: parse text template %s: %w", absPath, err)
		}
		textTmpls[key] = t
	}
	return nil
}

// RenderEmail renders the named template to HTML and plain text.
func (r *Renderer) RenderEmail(_ context.Context, templateName, layoutName string, props map[string]any) (string, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	html, err := r.renderHTML(templateName, layoutName, props)
	if err != nil {
		return "", "", err
	}

	text, err := r.renderText(templateName, layoutName, props)
	if err != nil {
		return "", "", err
	}

	return html, text, nil
}

func (r *Renderer) renderHTML(templateName, layoutName string, props map[string]any) (string, error) {
	contentTmpl, ok := r.htmlTmpls[templateName]
	if !ok {
		return "", fmt.Errorf("tmpl: unknown html template %q", templateName)
	}

	layoutTmpl, hasLayout := r.htmlLayouts[layoutName]
	if !hasLayout || layoutName == "" {
		// Render content directly without layout.
		var buf bytes.Buffer
		if err := contentTmpl.Execute(&buf, props); err != nil {
			return "", fmt.Errorf("tmpl: render html %s: %w", templateName, err)
		}
		return buf.String(), nil
	}

	// Clone layout and add content as a sub-template.
	combined, err := layoutTmpl.Clone()
	if err != nil {
		return "", fmt.Errorf("tmpl: clone html layout %s: %w", layoutName, err)
	}
	if _, err := combined.AddParseTree("content", contentTmpl.Tree); err != nil {
		return "", fmt.Errorf("tmpl: compose html %s with layout %s: %w", templateName, layoutName, err)
	}

	var buf bytes.Buffer
	if err := combined.ExecuteTemplate(&buf, "layout", props); err != nil {
		return "", fmt.Errorf("tmpl: render html %s: %w", templateName, err)
	}
	return buf.String(), nil
}

func (r *Renderer) renderText(templateName, layoutName string, props map[string]any) (string, error) {
	contentTmpl, ok := r.textTmpls[templateName]
	if !ok {
		// Text template is optional — return empty string.
		return "", nil
	}

	layoutTmpl, hasLayout := r.textLayouts[layoutName]
	if !hasLayout || layoutName == "" {
		var buf bytes.Buffer
		if err := contentTmpl.Execute(&buf, props); err != nil {
			return "", fmt.Errorf("tmpl: render text %s: %w", templateName, err)
		}
		return buf.String(), nil
	}

	combined, err := layoutTmpl.Clone()
	if err != nil {
		return "", fmt.Errorf("tmpl: clone text layout %s: %w", layoutName, err)
	}
	if _, err := combined.AddParseTree("content", contentTmpl.Tree); err != nil {
		return "", fmt.Errorf("tmpl: compose text %s with layout %s: %w", templateName, layoutName, err)
	}

	var buf bytes.Buffer
	if err := combined.ExecuteTemplate(&buf, "layout", props); err != nil {
		return "", fmt.Errorf("tmpl: render text %s: %w", templateName, err)
	}
	return buf.String(), nil
}

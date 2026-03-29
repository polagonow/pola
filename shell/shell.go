// Package shell renders the complete HTML document shell for Pola pages.
// It is renderer-agnostic: the only renderer-specific piece is the body
// content (e.g. `<div id="root"></div>` for React), which is supplied at
// construction time via New().
//
// The HTML template is written in templ (https://templ.guide/); run
// `go tool templ generate` after editing shell.templ to regenerate shell_templ.go.
package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/a-h/templ"
	"github.com/polagonow/pola/core"
)

// Shell implements core.HTMLShell using templ-generated components.
type Shell struct {
	innerHTML string
}

// New returns a Shell that injects innerHTML as the body content of the page.
// Pass the renderer-specific root element, e.g.:
//
//	shell.New(`<div id="root"></div>`)   // React
//	shell.New(`<div id="app"></div>`)    // Vue
//	shell.New(``)                        // HTMX (no root div needed)
func New(innerHTML string) *Shell {
	return &Shell{innerHTML: innerHTML}
}

// Render returns the complete HTML document for a page shell.
// It implements core.HTMLShell.
func (s *Shell) Render(p core.ShellParams) string {
	mergeDocumentMeta(&p)
	var buf strings.Builder
	_ = page(s.innerHTML, ImportMap(p.ImportURLs), p.Scripts, p.ClientScript, p.Stylesheets, p.Metadata, p.DocumentProps).
		Render(context.Background(), &buf)
	return buf.String()
}

// ImportMap generates a <script type="importmap"> block from a module-ID →
// chunk-URL map. Returns an empty string when importURLs is nil or empty.
func ImportMap(importURLs map[string]string) string {
	if len(importURLs) == 0 {
		return ""
	}
	payload, err := json.Marshal(map[string]any{"imports": importURLs})
	if err != nil {
		return ""
	}
	// json.Marshal escapes <, >, & to \uXXXX by default, so "</script>"
	// sequences in keys/values cannot break out of the script tag.
	return fmt.Sprintf(`<script type="importmap">%s</script>`, payload)
}

// styleBlock returns the embedded CSS wrapped in a <style> tag.
// Used from shell.templ because templ does not process expressions inside <style> content.
func styleBlock() templ.Component {
	return templ.Raw("<style>" + styles + "</style>")
}

// inlineScript wraps a trusted JS string in a <script> tag.
// Used from shell.templ because templ does not process expressions inside <script> content.
func inlineScript(sc string) templ.Component {
	return templ.Raw("<script>" + sc + "</script>")
}

// renderMetaBlock renders all metadata as HTML using the reflection-based renderer.
func renderMetaBlock(m *core.Metadata) templ.Component {
	return templ.Raw(core.RenderMetaHTML(m))
}

// charset returns the document charset, defaulting to "UTF-8".
func charset(dp *core.DocumentProps) string {
	if dp != nil && dp.Charset != "" {
		return dp.Charset
	}
	return "UTF-8"
}

// viewport returns the viewport meta content, defaulting to the standard responsive value.
func viewport(dp *core.DocumentProps) string {
	if dp != nil && dp.Viewport != "" {
		return dp.Viewport
	}
	return "width=device-width,initial-scale=1"
}

// mergeDocumentMeta applies DocumentProps overrides as fallbacks into Metadata.
// This deduplicates head elements: structured tags (title, description, etc.)
// extracted from the root layout are merged into the Metadata system rather
// than rendered as raw HTML alongside the framework's own meta tags.
func mergeDocumentMeta(params *core.ShellParams) {
	dp := params.DocumentProps
	if dp == nil {
		return
	}
	m := params.Metadata
	if m == nil {
		m = &core.Metadata{}
		params.Metadata = m
	}
	if dp.Title != "" && core.ResolveTitle(m.Title) == "" {
		m.Title = core.Title{Default: dp.Title}
	}
	for name, content := range dp.MetaOverrides {
		c := content // capture for pointer
		switch name {
		case "description":
			if m.Description == nil {
				m.Description = &c
			}
		case "generator":
			if m.Generator == nil {
				m.Generator = &c
			}
		case "creator":
			if m.Creator == nil {
				m.Creator = &c
			}
		case "publisher":
			if m.Publisher == nil {
				m.Publisher = &c
			}
		case "referrer":
			if m.Referrer == nil {
				m.Referrer = &c
			}
		case "application-name":
			if m.ApplicationName == nil {
				m.ApplicationName = &c
			}
		default:
			if m.Other == nil {
				m.Other = make(map[string]string)
			}
			if _, exists := m.Other[name]; !exists {
				m.Other[name] = c
			}
		}
	}
}

// htmlAttrs returns the templ.Attributes for the <html> element.
// Defaults lang to "en" when not specified.
func htmlAttrs(dp *core.DocumentProps) templ.Attributes {
	attrs := templ.Attributes{"lang": "en"}
	if dp != nil {
		for k, v := range dp.HTMLAttributes {
			attrs[k] = v
		}
	}
	return attrs
}

// bodyAttrs returns the templ.Attributes for the <body> element.
func bodyAttrs(dp *core.DocumentProps) templ.Attributes {
	if dp == nil || len(dp.BodyAttributes) == 0 {
		return nil
	}
	attrs := make(templ.Attributes, len(dp.BodyAttributes))
	for k, v := range dp.BodyAttributes {
		attrs[k] = v
	}
	return attrs
}

// customHeadElements renders extra <head> elements extracted from the root layout.
func customHeadElements(dp *core.DocumentProps) templ.Component {
	if dp == nil || len(dp.HeadElements) == 0 {
		return templ.NopComponent
	}
	return templ.Raw(strings.Join(dp.HeadElements, "\n"))
}

// bodyPrefix returns the HTML to render before the root div.
func bodyPrefix(dp *core.DocumentProps) string {
	if dp == nil {
		return ""
	}
	return dp.BodyPrefix
}

// bodySuffix returns the HTML to render after the scripts.
func bodySuffix(dp *core.DocumentProps) string {
	if dp == nil {
		return ""
	}
	return dp.BodySuffix
}

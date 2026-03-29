// Package shell renders the complete HTML document shell for Pola pages.
// It is renderer-agnostic: the only renderer-specific piece is the body
// content (e.g. `<div id="root"></div>` for React), which is supplied at
// construction time via New().
package shell

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/polagonow/pola/core"
)

// Shell implements core.HTMLShell using Go html/template.
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

	scripts := make([]template.JS, len(p.Scripts))
	for i, sc := range p.Scripts {
		scripts[i] = template.JS(sc)
	}

	d := &pageData{
		HTMLOpen:           template.HTML("<html" + string(renderAttrs(htmlAttrsMap(p.DocumentProps))) + ">"),
		Charset:            charset(p.DocumentProps),
		Viewport:           viewport(p.DocumentProps),
		MetaBlock:          template.HTML(core.RenderMetaHTML(p.Metadata)),
		CustomHeadElements: customHeadElements(p.DocumentProps),
		HasStylesheets:     len(p.Stylesheets) > 0,
		StyleBlock:         template.HTML("<style>" + styles + "</style>"),
		Stylesheets:        p.Stylesheets,
		ImportMap:          template.HTML(ImportMap(p.ImportURLs)),
		BodyOpen:           template.HTML("<body" + string(renderAttrs(bodyAttrsMap(p.DocumentProps))) + ">"),
		BodyPrefix:         template.HTML(bodyPrefix(p.DocumentProps)),
		InnerHTML:          template.HTML(s.innerHTML),
		Scripts:            scripts,
		ClientScript:       p.ClientScript,
		BodySuffix:         template.HTML(bodySuffix(p.DocumentProps)),
	}

	html, err := renderPage(d)
	if err != nil {
		return "<!-- shell render error: " + err.Error() + " -->"
	}
	return html
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

// htmlAttrsMap returns the attribute map for the <html> element.
// Defaults lang to "en" when not specified.
func htmlAttrsMap(dp *core.DocumentProps) map[string]string {
	attrs := map[string]string{"lang": "en"}
	if dp != nil {
		for k, v := range dp.HTMLAttributes {
			attrs[k] = v
		}
	}
	return attrs
}

// bodyAttrsMap returns the attribute map for the <body> element.
func bodyAttrsMap(dp *core.DocumentProps) map[string]string {
	if dp == nil || len(dp.BodyAttributes) == 0 {
		return nil
	}
	return dp.BodyAttributes
}

// renderAttrs formats a map as HTML element attributes (e.g. ` lang="en" dir="ltr"`).
func renderAttrs(attrs map[string]string) template.HTML {
	if len(attrs) == 0 {
		return ""
	}
	var buf strings.Builder
	for k, v := range attrs {
		buf.WriteString(" ")
		buf.WriteString(k)
		buf.WriteString(`="`)
		buf.WriteString(template.HTMLEscapeString(v))
		buf.WriteString(`"`)
	}
	return template.HTML(buf.String())
}

// customHeadElements renders extra <head> elements extracted from the root layout.
func customHeadElements(dp *core.DocumentProps) template.HTML {
	if dp == nil || len(dp.HeadElements) == 0 {
		return ""
	}
	return template.HTML(strings.Join(dp.HeadElements, "\n"))
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

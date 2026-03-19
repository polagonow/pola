// Package shell renders the complete HTML document shell for Pola pages.
// It is renderer-agnostic: the only renderer-specific piece is the body
// content (e.g. `<div id="root"></div>` for React), which is supplied at
// construction time via New().
package shell

import (
	"encoding/json"
	"fmt"
	"html/template"
	"slices"
	"strings"

	"github.com/polagonow/pola/core"
)

// Shell implements core.HTMLShell using the built-in Go HTML template.
type Shell struct {
	innerHTML template.HTML
}

// New returns a Shell that injects innerHTML as the body content of the page.
// Pass the renderer-specific root element, e.g.:
//
//	shell.New(`<div id="root"></div>`)   // React
//	shell.New(`<div id="app"></div>`)    // Vue
//	shell.New(``)                        // HTMX (no root div needed)
func New(innerHTML template.HTML) *Shell {
	return &Shell{innerHTML: innerHTML}
}

// shellData is the typed data passed to shellTmpl.
type shellData struct {
	InnerHTML    template.HTML
	ImportMap    template.HTML
	HeadMeta     template.HTML
	Scripts      []template.JS
	ClientScript template.URL
}

var shellTmpl = template.Must(template.New("shell").Parse(shellTemplate))

// Render returns the complete HTML document for a page shell.
// It implements core.HTMLShell.
func (s *Shell) Render(p core.ShellParams) string {
	data := shellData{
		InnerHTML:    s.innerHTML,
		ImportMap:    template.HTML(ImportMap(p.ImportURLs)),
		HeadMeta:     renderHeadMeta(p.Metadata),
		ClientScript: template.URL(p.ClientScript),
	}
	for _, sc := range p.Scripts {
		data.Scripts = append(data.Scripts, template.JS(sc))
	}
	var buf strings.Builder
	_ = shellTmpl.Execute(&buf, data)
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
	return fmt.Sprintf(`<script type="importmap">%s</script>`, payload)
}

// resolveTitle returns the string to use inside <title>, applying NextJS-style
// template substitution (%s → page title). Returns "" when all fields are empty.
func resolveTitle(t core.Title) string {
	if t.Absolute != "" {
		return t.Absolute
	}
	if t.Default != "" && t.Template != "" {
		return strings.Replace(t.Template, "%s", t.Default, 1)
	}
	if t.Default != "" {
		return t.Default
	}
	return t.Template
}

// renderRobotsContent builds the comma-separated robots directive string.
func renderRobotsContent(r *core.Robots) string {
	var parts []string

	addBool := func(trueVal, falseVal string, v *bool) {
		if v == nil {
			return
		}
		if *v {
			parts = append(parts, trueVal)
		} else {
			parts = append(parts, falseVal)
		}
	}
	addNoOnly := func(directive string, v *bool) {
		if v != nil && *v {
			parts = append(parts, directive)
		}
	}

	addBool("index", "noindex", r.Index)
	addBool("follow", "nofollow", r.Follow)
	addNoOnly("noarchive", r.NoArchive)
	addNoOnly("nosnippet", r.NoSnippet)
	addNoOnly("noimageindex", r.NoImageIndex)
	addNoOnly("nocache", r.NoCache)
	addNoOnly("notranslate", r.NoTranslate)

	if r.MaxSnippet != nil {
		parts = append(parts, fmt.Sprintf("max-snippet:%d", *r.MaxSnippet))
	}
	if r.MaxImagePreview != nil {
		parts = append(parts, fmt.Sprintf("max-image-preview:%s", *r.MaxImagePreview))
	}
	if r.MaxVideoPreview != nil {
		parts = append(parts, fmt.Sprintf("max-video-preview:%d", *r.MaxVideoPreview))
	}

	return strings.Join(parts, ", ")
}

// renderHeadMeta converts *core.Metadata into a sequence of safe HTML head
// tags. All user-supplied values are escaped via template.HTMLEscapeString
// before being written; the result is then wrapped in template.HTML so the
// template engine does not double-escape it.
func renderHeadMeta(m *core.Metadata) template.HTML {
	if m == nil {
		return ""
	}

	var b strings.Builder
	esc := template.HTMLEscapeString

	writeMeta := func(name, content string) {
		if content != "" {
			fmt.Fprintf(&b, "  <meta name=%q content=%q>\n", name, content)
		}
	}
	writeProp := func(prop, content string) {
		if content != "" {
			fmt.Fprintf(&b, "  <meta property=%q content=%q>\n", prop, content)
		}
	}

	// --- <title> ---
	if t := resolveTitle(m.Title); t != "" {
		fmt.Fprintf(&b, "  <title>%s</title>\n", esc(t))
	}

	// --- Standard <meta name="..."> tags ---
	if m.Description != nil {
		writeMeta("description", *m.Description)
	}
	if m.ApplicationName != nil {
		writeMeta("application-name", *m.ApplicationName)
	}
	if m.Generator != nil {
		writeMeta("generator", *m.Generator)
	}
	if m.Referrer != nil {
		writeMeta("referrer", *m.Referrer)
	}
	if m.Creator != nil {
		writeMeta("creator", *m.Creator)
	}
	if m.Publisher != nil {
		writeMeta("publisher", *m.Publisher)
	}
	if len(m.Keywords) > 0 {
		writeMeta("keywords", strings.Join(m.Keywords, ", "))
	}

	// --- Authors ---
	for _, a := range m.Authors {
		if a.URL != nil {
			fmt.Fprintf(&b, "  <link rel=\"author\" href=%q>\n", esc(*a.URL))
		}
		if a.Name != nil {
			writeMeta("author", *a.Name)
		}
	}

	// --- Robots ---
	if m.Robots != nil {
		if rc := renderRobotsContent(m.Robots); rc != "" {
			writeMeta("robots", rc)
		}
	}

	// --- Canonical + hreflang alternates ---
	if m.Alternates != nil {
		if m.Alternates.Canonical != nil {
			fmt.Fprintf(&b, "  <link rel=\"canonical\" href=%q>\n", esc(*m.Alternates.Canonical))
		}
		langs := make([]string, 0, len(m.Alternates.Languages))
		for lang := range m.Alternates.Languages {
			langs = append(langs, lang)
		}
		slices.Sort(langs)
		for _, lang := range langs {
			fmt.Fprintf(&b, "  <link rel=\"alternate\" hreflang=%q href=%q>\n",
				esc(lang), esc(m.Alternates.Languages[lang]))
		}
	}

	// --- Icons ---
	if m.Icons != nil {
		writeIconLinks := func(rel string, icons []core.Icon) {
			for _, ic := range icons {
				tag := fmt.Sprintf("  <link rel=%q href=%q", esc(rel), esc(ic.URL))
				if ic.Sizes != nil {
					tag += fmt.Sprintf(" sizes=%q", esc(*ic.Sizes))
				}
				if ic.Type != nil {
					tag += fmt.Sprintf(" type=%q", esc(*ic.Type))
				}
				b.WriteString(tag + ">\n")
			}
		}
		writeIconLinks("icon", m.Icons.Icon)
		writeIconLinks("shortcut icon", m.Icons.Shortcut)
		writeIconLinks("apple-touch-icon", m.Icons.Apple)
		for _, oi := range m.Icons.Other {
			writeIconLinks(oi.Rel, []core.Icon{oi.Icon})
		}
	}

	// --- Open Graph ---
	if m.OpenGraph != nil {
		og := m.OpenGraph
		ogType := "website"
		if og.Type != nil {
			ogType = *og.Type
		}
		writeProp("og:type", ogType)
		if og.Title != nil {
			writeProp("og:title", *og.Title)
		}
		if og.Description != nil {
			writeProp("og:description", *og.Description)
		}
		if og.URL != nil {
			writeProp("og:url", *og.URL)
		}
		if og.SiteName != nil {
			writeProp("og:site_name", *og.SiteName)
		}
		if og.Locale != nil {
			writeProp("og:locale", *og.Locale)
		}
		for _, img := range og.Images {
			writeProp("og:image", img.URL)
			if img.SecureURL != nil {
				writeProp("og:image:secure_url", *img.SecureURL)
			}
			if img.Type != nil {
				writeProp("og:image:type", *img.Type)
			}
			if img.Alt != nil {
				writeProp("og:image:alt", *img.Alt)
			}
			if img.Width != nil {
				writeProp("og:image:width", fmt.Sprint(*img.Width))
			}
			if img.Height != nil {
				writeProp("og:image:height", fmt.Sprint(*img.Height))
			}
		}
	}

	// --- Twitter ---
	if m.Twitter != nil {
		tw := m.Twitter
		if tw.Card != nil {
			writeMeta("twitter:card", *tw.Card)
		}
		if tw.Site != nil {
			writeMeta("twitter:site", *tw.Site)
		}
		if tw.Creator != nil {
			writeMeta("twitter:creator", *tw.Creator)
		}
		if tw.Title != nil {
			writeMeta("twitter:title", *tw.Title)
		}
		if tw.Description != nil {
			writeMeta("twitter:description", *tw.Description)
		}
		for _, img := range tw.Images {
			writeMeta("twitter:image", img)
		}
	}

	// --- Verification ---
	if m.Verification != nil {
		v := m.Verification
		if v.Google != nil {
			writeMeta("google-site-verification", *v.Google)
		}
		if v.Yandex != nil {
			writeMeta("yandex-verification", *v.Yandex)
		}
		if v.Yahoo != nil {
			writeMeta("y_key", *v.Yahoo)
		}
		if v.Me != nil {
			fmt.Fprintf(&b, "  <link rel=\"me\" href=%q>\n", esc(*v.Me))
		}
	}

	// --- Custom meta tags ---
	if len(m.Other) > 0 {
		keys := make([]string, 0, len(m.Other))
		for k := range m.Other {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			writeMeta(k, m.Other[k])
		}
	}

	return template.HTML(b.String()) //nolint:gosec // content is HTML-escaped above
}

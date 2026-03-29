package shell

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed page.tmpl
var pageTmplFS embed.FS

var pageTmpl = template.Must(template.New("page.tmpl").ParseFS(pageTmplFS, "page.tmpl"))

// pageData holds all values passed to the page template.
type pageData struct {
	HTMLOpen           template.HTML // e.g. `<html lang="en">`
	Charset            string
	Viewport           string
	MetaBlock          template.HTML
	CustomHeadElements template.HTML
	HasStylesheets     bool
	StyleBlock         template.HTML
	Stylesheets        []string
	ImportMap          template.HTML
	BodyOpen           template.HTML // e.g. `<body>` or `<body class="dark">`
	BodyPrefix         template.HTML
	InnerHTML          template.HTML
	Scripts            []template.JS
	ClientScript       string
	BodySuffix         template.HTML
}

// renderPage executes the page template with the given data and returns the HTML string.
func renderPage(d *pageData) (string, error) {
	var buf strings.Builder
	if err := pageTmpl.Execute(&buf, d); err != nil {
		return "", fmt.Errorf("shell: render page: %w", err)
	}
	return buf.String(), nil
}

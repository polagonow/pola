// Package shell renders the complete HTML document shell for Pola React
// server-rendered pages.
package shell

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/polagonow/pola/core"
)

// Shell implements core.HTMLShell using the built-in Go HTML template.
type Shell struct{}

// New returns a Shell with default options.
func New() *Shell { return &Shell{} }

// shellData is the typed data passed to shellTmpl.
type shellData struct {
	ImportMap    template.HTML
	Scripts      []template.JS
	ClientScript template.URL
}

var shellTmpl = template.Must(template.New("shell").Parse(shellTemplate))

// Render returns the complete HTML document for a page shell.
// It implements core.HTMLShell.
func (s *Shell) Render(p core.ShellParams) string {
	data := shellData{
		ImportMap:    template.HTML(ImportMap(p.ImportURLs)),
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

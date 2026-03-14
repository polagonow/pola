// Package html renders the complete HTML shell for GoJSX server-rendered pages.
//
// Usage:
//
//	params := html.Params{ImportURLs: ..., ClientScript: ...}
//	// optionally append inline scripts:
//	params.Scripts = append(params.Scripts, "self.__flight_data="+flightJSON)
//	fmt.Fprint(w, html.Render(params))
package html

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

// Params holds the dynamic values needed to render the HTML shell.
type Params struct {
	// ImportURLs maps RSC module IDs to their hashed browser chunk URLs,
	// used to generate the <script type="importmap"> block.
	ImportURLs map[string]string

	// ClientScript is the URL of the compiled client entry module
	// (e.g. "/public/assets/_client-HASH.js").
	ClientScript string

	// Scripts holds bare JS expressions to embed as inline <script> blocks
	// before the client module tag (e.g. "self.__flight_data=...").
	Scripts []string
}

// shellData is the typed data passed to shellTmpl.
// html/template uses the types to apply correct context-aware escaping.
type shellData struct {
	ImportMap    template.HTML // trusted HTML — emitted verbatim
	Scripts      []template.JS // trusted JS — safe in <script> context
	ClientScript template.URL  // trusted URL — safe in src attribute context
}

var shellTmpl = template.Must(template.New("shell").Parse(shellTemplate))

// Render returns the complete HTML document for a page shell.
// Inline Scripts (if any) are emitted as <script> blocks before the client module.
func Render(p Params) string {
	data := shellData{
		ImportMap:    template.HTML(ImportMap(p.ImportURLs)),
		ClientScript: template.URL(p.ClientScript),
	}
	for _, s := range p.Scripts {
		data.Scripts = append(data.Scripts, template.JS(s))
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

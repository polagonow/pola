// Package html renders the HTML shell for GoJSX server-rendered pages.
//
// The shell is emitted in two parts to support streaming RSC:
//
//	Open(p)   — writes <!DOCTYPE html>…<body> including the import map
//	Close(p)  — writes the client <script> tag and </body></html>
//
// RSC flight data (or inline flight JSON) is written between the two calls.
package html

import (
	"encoding/json"
	"fmt"
)

// Params holds the dynamic values needed to render the HTML shell.
type Params struct {
	// ImportURLs maps RSC module IDs to their hashed browser chunk URLs,
	// used to generate the <script type="importmap"> block.
	ImportURLs map[string]string

	// ClientScript is the URL of the compiled client entry module
	// (e.g. "/public/assets/_client-HASH.js").
	ClientScript string
}

// Open returns the HTML head and opening body content, including the
// generated import map. Write this before streaming RSC flight data.
func Open(p Params) string {
	return shellHead + ImportMap(p.ImportURLs) + shellBody
}

// Close returns the client <script type="module"> tag followed by
// </body></html>. Write this after all RSC flight data has been sent.
// Falls back to "/public/_client.js" when ClientScript is empty.
func Close(p Params) string {
	return fmt.Sprintf(`  <script type="module" src="%s"></script>
</body>
</html>`, p.ClientScript)
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

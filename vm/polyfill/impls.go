// Package polyfill contains the shared JavaScript polyfill files and a loader
// that evaluates them in order into any script-based JS VM.
package polyfill

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed *.js
var Files embed.FS

// ScriptRunner is implemented by each VM's thin adapter to evaluate JS source.
type ScriptRunner interface {
	RunScript(src, name string) error
}

// Load evaluates every embedded .js file into r, in filename order.
func Load(r ScriptRunner) error {
	return fs.WalkDir(Files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".js") {
			return nil
		}
		data, err := fs.ReadFile(Files, path)
		if err != nil {
			return fmt.Errorf("polyfill: read %s: %w", path, err)
		}
		return r.RunScript(string(data), path)
	})
}

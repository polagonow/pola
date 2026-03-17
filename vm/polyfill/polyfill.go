// Package polyfill contains the shared JavaScript polyfill files and a loader
// that evaluates them in order into any script-based JS VM.
//
// Files are evaluated in filename order (01_, 02_, …) so dependencies are
// always satisfied:
//
//	01_microtask.js        — queueMicrotask / __drainMicrotasks__
//	02_textencoding.js     — TextEncoder / TextDecoder
//	03_messagechannel.js   — MessageChannel  (needs 01)
//	04_readablestream.js   — ReadableStream / __pullStream__  (needs 01)
//	05_webpackrequire.js   — __webpack_require__ stubs
//	06_abortcontroller.js  — AbortSignal / AbortController
//	07_promise.js          — Promise (pure-JS, routed through queueMicrotask)
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

// LoadAll evaluates every embedded .js file into r, in filename order.
func LoadAll(r ScriptRunner) error {
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

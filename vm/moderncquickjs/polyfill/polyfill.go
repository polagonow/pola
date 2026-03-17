// Package polyfill provides Web API polyfills for the modernc.org/quickjs VM.
//
// In addition to the shared polyfills, promise.js replaces the native Promise
// with a queueMicrotask-backed implementation. modernc.org/quickjs exposes no
// JS_ExecutePendingJob API, so native Promise continuations would never fire.
package polyfill

import (
	"embed"
	"fmt"
	"io/fs"

	mquickjs "modernc.org/quickjs"

	"gojsx/vm/polyfill"
)

//go:embed promise.js
var extraFiles embed.FS

type runner struct{ vm *mquickjs.VM }

func (r *runner) RunScript(src, name string) error {
	if _, err := r.vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("polyfill %s: %w", name, err)
	}
	return nil
}

// Enable installs all polyfills into vm.
// Must be called after basic globals are set and before the bundle runs.
func Enable(vm *mquickjs.VM) error {
	r := &runner{vm}
	if err := polyfill.LoadAll(r); err != nil {
		return err
	}
	// Load promise.js after microtask so queueMicrotask is already defined.
	data, err := fs.ReadFile(extraFiles, "promise.js")
	if err != nil {
		return fmt.Errorf("polyfill promise.js: %w", err)
	}
	return r.RunScript(string(data), "promise.js")
}

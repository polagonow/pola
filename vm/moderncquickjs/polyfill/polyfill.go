// Package polyfill provides Web API polyfills for the modernc.org/quickjs VM.
package polyfill

import (
	"fmt"

	mquickjs "modernc.org/quickjs"

	"gojsx/vm/polyfill"
)

type runner struct{ vm *mquickjs.VM }

func (r *runner) RunScript(src, name string) error {
	if _, err := r.vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("polyfill %s: %w", name, err)
	}
	return nil
}

// Enable installs all polyfills into vm.
func Enable(vm *mquickjs.VM) error {
	return polyfill.Load(&runner{vm})
}

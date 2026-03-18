// Package polyfill provides Web API polyfills for the QuickJS VM.
package polyfill

import (
	"fmt"

	quickjs "github.com/buke/quickjs-go"

	"github.com/polagonow/pola/vm/polyfill"
)

type runner struct{ ctx *quickjs.Context }

func (r *runner) RunScript(src, name string) error {
	ret := r.ctx.Eval(src, quickjs.EvalFileName(name))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("polyfill %s: %w", name, r.ctx.Exception())
	}
	return nil
}

// Enable installs all polyfills into ctx.
// Must be called after basic globals are set and before the bundle runs.
func Enable(ctx *quickjs.Context) error {
	return polyfill.Load(&runner{ctx})
}

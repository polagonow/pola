// Package polyfill provides Web API polyfills for the qjs VM.
package polyfill

import (
	"fmt"

	qjs "github.com/fastschema/qjs"

	"github.com/polagonow/pola/vm/polyfill"
)

type runner struct{ ctx *qjs.Context }

func (r *runner) RunScript(src, name string) error {
	val, err := r.ctx.Eval(name, qjs.Code(src))
	if val != nil {
		val.Free()
	}
	if err != nil {
		return fmt.Errorf("polyfill %s: %w", name, err)
	}
	return nil
}

// Enable installs all polyfills into ctx.
// Must be called after basic globals are set and before the bundle runs.
func Enable(ctx *qjs.Context) error {
	return polyfill.Load(&runner{ctx})
}

// Package polyfill provides Web API polyfills for the V8 VM.
package polyfill

import (
	"fmt"

	v8 "rogchap.com/v8go"

	"github.com/polagonow/pola/vm/polyfill"
)

type runner struct{ ctx *v8.Context }

func (r *runner) RunScript(src, name string) error {
	if _, err := r.ctx.RunScript(src, name); err != nil {
		return fmt.Errorf("polyfill %s: %w", name, err)
	}
	return nil
}

// Enable installs all polyfills into ctx.
// Must be called after basic globals are set and before the bundle runs.
func Enable(ctx *v8.Context) error {
	return polyfill.Load(&runner{ctx})
}

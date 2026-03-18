// Package polyfill provides Web API polyfills for the Goja VM.
package polyfill

import (
	"fmt"

	gojalib "github.com/dop251/goja"

	"github.com/polagonow/pola/framework"
	"github.com/polagonow/pola/vm/polyfill"
)

type runner struct{ rt *gojalib.Runtime }

func (r *runner) RunScript(src, name string) error {
	if _, err := r.rt.RunString(src); err != nil {
		return fmt.Errorf("polyfill %s: %w", name, err)
	}
	return nil
}

// Enable installs all polyfills onto rt as globals.
// Must be called before rt.RunProgram(serverBundle).
func Enable(rt *gojalib.Runtime) error {
	return polyfill.Load(&runner{rt})
}

// GojaVMInitContext implements framework.VMInitContext for Goja runtimes.
type GojaVMInitContext struct {
	Rt *gojalib.Runtime
}

// SetGlobal sets a named global in the Goja runtime.
func (c *GojaVMInitContext) SetGlobal(name string, value any) error {
	c.Rt.Set(name, value) //nolint:errcheck
	return nil
}

// RunScript executes a JavaScript snippet in the Goja runtime.
func (c *GojaVMInitContext) RunScript(source string) error {
	_, err := c.Rt.RunString(source)
	return err
}

// GojaPolyfillRegistry implements framework.PolyfillRegistry for Goja.
type GojaPolyfillRegistry struct{}

// Install installs all Goja polyfills onto the runtime provided by ctx.
func (r *GojaPolyfillRegistry) Install(ctx framework.VMInitContext) error {
	gojaCtx, ok := ctx.(*GojaVMInitContext)
	if !ok {
		return fmt.Errorf("polyfill: GojaPolyfillRegistry requires *GojaVMInitContext, got %T", ctx)
	}
	return Enable(gojaCtx.Rt)
}

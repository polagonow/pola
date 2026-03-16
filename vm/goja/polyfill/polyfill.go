// Package polyfill provides native Go implementations of Web APIs required by
// react-server-dom-webpack/server.browser inside the Goja VM.
//
// Call Enable(rt) once per VM after setting up basic globals (global,
// globalThis, etc.) and before running the compiled server bundle.
//
// Polyfills (in dependency order):
//
//   - microtask       — queueMicrotask / __microtaskQueue__ / __drainMicrotasks__
//   - textencoding    — TextEncoder / TextDecoder
//   - messagechannel  — MessageChannel  (depends on microtask)
//   - readablestream  — ReadableStream / __pullStream__  (depends on microtask)
//   - webpackrequire  — __webpack_require__ / __webpack_chunk_load__
//   - abortcontroller — AbortController / AbortSignal
package polyfill

import (
	"fmt"

	"gojsx/framework"

	"github.com/dop251/goja"

	"gojsx/vm/goja/polyfill/abortcontroller"
	"gojsx/vm/goja/polyfill/messagechannel"
	"gojsx/vm/goja/polyfill/microtask"
	"gojsx/vm/goja/polyfill/readablestream"
	"gojsx/vm/goja/polyfill/textencoding"
	"gojsx/vm/goja/polyfill/webpackrequire"
)

// Enable installs all polyfills onto rt as globals.
// Must be called before rt.RunProgram(serverBundle).
func Enable(rt *goja.Runtime) {
	polyfills := []func(*goja.Runtime){
		microtask.Enable,       // foundation: must run first
		textencoding.Enable,    // standalone
		messagechannel.Enable,  // depends on __microtaskQueue__
		readablestream.Enable,  // depends on __drainMicrotasks__
		webpackrequire.Enable,  // standalone
		abortcontroller.Enable, // standalone
	}

	for _, enable := range polyfills {
		enable(rt)
	}
}

// GojaVMInitContext implements framework.VMInitContext for Goja runtimes.
type GojaVMInitContext struct {
	Rt *goja.Runtime
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
	Enable(gojaCtx.Rt)
	return nil
}

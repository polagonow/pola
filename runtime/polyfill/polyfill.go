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
	"github.com/dop251/goja"

	"gojsx/runtime/polyfill/abortcontroller"
	"gojsx/runtime/polyfill/messagechannel"
	"gojsx/runtime/polyfill/microtask"
	"gojsx/runtime/polyfill/readablestream"
	"gojsx/runtime/polyfill/textencoding"
	"gojsx/runtime/polyfill/webpackrequire"
)

// Enable installs all polyfills onto rt as globals.
// Must be called before rt.RunProgram(serverBundle).
func Enable(rt *goja.Runtime) {
	microtask.Enable(rt)       // foundation: must run first
	textencoding.Enable(rt)    // standalone
	messagechannel.Enable(rt)  // depends on __microtaskQueue__
	readablestream.Enable(rt)  // depends on __drainMicrotasks__
	webpackrequire.Enable(rt)  // standalone
	abortcontroller.Enable(rt) // standalone
}

// Package polyfill provides native Go implementations of Web APIs required by
// react-server-dom-webpack/server.browser inside the Sobek VM.
//
// Call Enable(rt) once per VM after setting up basic globals and before running
// the compiled server bundle.
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
	sobeklib "github.com/grafana/sobek"

	"gojsx/vm/sobek/polyfill/abortcontroller"
	"gojsx/vm/sobek/polyfill/messagechannel"
	"gojsx/vm/sobek/polyfill/microtask"
	"gojsx/vm/sobek/polyfill/readablestream"
	"gojsx/vm/sobek/polyfill/textencoding"
	"gojsx/vm/sobek/polyfill/webpackrequire"
)

// Enable installs all polyfills onto rt as globals.
// Must be called before rt.RunProgram(serverBundle).
func Enable(rt *sobeklib.Runtime) {
	polyfills := []func(*sobeklib.Runtime){
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

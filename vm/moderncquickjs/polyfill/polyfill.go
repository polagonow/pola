// Package polyfill provides Web API polyfills for the modernc.org/quickjs VM.
//
// Installation order matters:
//
//	microtask       — must run first (provides queueMicrotask / __drainMicrotasks__)
//	promise         — replaces native Promise; depends on queueMicrotask
//	textencoding    — standalone
//	messagechannel  — depends on queueMicrotask
//	readablestream  — depends on __drainMicrotasks__
//	webpackrequire  — standalone
//	abortcontroller — standalone
package polyfill

import (
	mquickjs "modernc.org/quickjs"

	"gojsx/vm/moderncquickjs/polyfill/abortcontroller"
	"gojsx/vm/moderncquickjs/polyfill/messagechannel"
	"gojsx/vm/moderncquickjs/polyfill/microtask"
	"gojsx/vm/moderncquickjs/polyfill/promise"
	"gojsx/vm/moderncquickjs/polyfill/readablestream"
	"gojsx/vm/moderncquickjs/polyfill/textencoding"
	"gojsx/vm/moderncquickjs/polyfill/webpackrequire"
)

// Enable installs all polyfills into vm.
// Must be called after basic globals are set and before the bundle runs.
func Enable(vm *mquickjs.VM) error {
	polyfills := []func(*mquickjs.VM) error{
		microtask.Enable,
		promise.Enable,
		textencoding.Enable,
		messagechannel.Enable,
		readablestream.Enable,
		webpackrequire.Enable,
		abortcontroller.Enable,
	}

	for _, enable := range polyfills {
		if err := enable(vm); err != nil {
			return err
		}
	}
	return nil
}

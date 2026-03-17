// Package polyfill provides Web API polyfills for the qjs VM.
//
// Installation order matters:
//
//	microtask       — must run first (provides queueMicrotask / __drainMicrotasks__)
//	textencoding    — standalone
//	messagechannel  — depends on queueMicrotask
//	readablestream  — depends on __drainMicrotasks__
//	webpackrequire  — standalone
//	abortcontroller — standalone
package polyfill

import (
	qjs "github.com/fastschema/qjs"

	"gojsx/vm/qjs/polyfill/abortcontroller"
	"gojsx/vm/qjs/polyfill/messagechannel"
	"gojsx/vm/qjs/polyfill/microtask"
	"gojsx/vm/qjs/polyfill/readablestream"
	"gojsx/vm/qjs/polyfill/textencoding"
	"gojsx/vm/qjs/polyfill/webpackrequire"
)

// Enable installs all polyfills into ctx.
// Must be called after basic globals are set and before the bundle runs.
func Enable(ctx *qjs.Context) error {
	polyfills := []func(*qjs.Context) error{
		microtask.Enable,
		textencoding.Enable,
		messagechannel.Enable,
		readablestream.Enable,
		webpackrequire.Enable,
		abortcontroller.Enable,
	}

	for _, enable := range polyfills {
		if err := enable(ctx); err != nil {
			return err
		}
	}
	return nil
}

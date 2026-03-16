// Package polyfill provides Web API polyfills for the V8 VM.
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
	v8 "rogchap.com/v8go"

	"gojsx/vm/v8go/polyfill/abortcontroller"
	"gojsx/vm/v8go/polyfill/messagechannel"
	"gojsx/vm/v8go/polyfill/microtask"
	"gojsx/vm/v8go/polyfill/readablestream"
	"gojsx/vm/v8go/polyfill/textencoding"
	"gojsx/vm/v8go/polyfill/webpackrequire"
)

// Enable installs all polyfills into ctx.
// Must be called after basic globals are set and before the bundle runs.
func Enable(ctx *v8.Context) error {
	polyfills := []func(*v8.Context) error{
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

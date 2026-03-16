// Package polyfill provides Web API polyfills for the QuickJS VM.
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
	quickjs "github.com/buke/quickjs-go"

	"gojsx/vm/quickjsgo/polyfill/abortcontroller"
	"gojsx/vm/quickjsgo/polyfill/messagechannel"
	"gojsx/vm/quickjsgo/polyfill/microtask"
	"gojsx/vm/quickjsgo/polyfill/readablestream"
	"gojsx/vm/quickjsgo/polyfill/textencoding"
	"gojsx/vm/quickjsgo/polyfill/webpackrequire"
)

// Enable installs all polyfills into ctx.
// Must be called after basic globals are set and before the bundle runs.
func Enable(ctx *quickjs.Context) error {
	polyfills := []func(*quickjs.Context) error{
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

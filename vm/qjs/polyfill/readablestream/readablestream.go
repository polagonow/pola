// Package readablestream provides ReadableStream and __pullStream__ for the
// QuickJS VM.
//
// Requires the microtask package to have been enabled first.
package readablestream

import (
	"fmt"

	qjs "github.com/fastschema/qjs"
)

const src = `
(function() {
	function ReadableStreamController() {
		this._chunks = [];
		this._closed = false;
		this._error = null;
	}
	ReadableStreamController.prototype.enqueue = function(chunk) {
		if (this._closed) return;
		this._chunks.push(chunk);
	};
	ReadableStreamController.prototype.close = function() {
		this._closed = true;
	};
	ReadableStreamController.prototype.error = function(err) {
		this._error = err;
		this._closed = true;
	};
	Object.defineProperty(ReadableStreamController.prototype, 'byobRequest', {
		get: function() { return null; },
		configurable: true
	});
	Object.defineProperty(ReadableStreamController.prototype, 'desiredSize', {
		get: function() { return this._closed ? 0 : 1; },
		configurable: true
	});

	function ReadableStream(src) {
		this._src = src || {};
		this._controller = new ReadableStreamController();
		this._started = false;
	}
	ReadableStream.prototype._start = function() {
		if (this._started) return;
		this._started = true;
		if (typeof this._src.start === 'function') {
			this._src.start(this._controller);
		}
	};
	ReadableStream.prototype._pull = function() {
		if (typeof this._src.pull === 'function') {
			this._src.pull(this._controller);
		}
	};

	globalThis.ReadableStream = ReadableStream;

	globalThis.__pullStream__ = function(s) {
		s._start();
		__drainMicrotasks__();
		s._pull();
		__drainMicrotasks__();
		var chunks = s._controller._chunks.splice(0);
		var closed = s._controller._closed;
		return { chunks: chunks, done: closed && chunks.length === 0 };
	};
})();
`

// Enable installs ReadableStream and __pullStream__ as globals into ctx.
// microtask.Enable must be called first.
func Enable(ctx *qjs.Context) error {
	val, err := ctx.Eval("readablestream.js", qjs.Code(src))
	if val != nil {
		val.Free()
	}
	if err != nil {
		return fmt.Errorf("readablestream: %w", err)
	}
	return nil
}

// Package abortcontroller provides AbortController and AbortSignal for the
// QuickJS VM.
package abortcontroller

import (
	"fmt"

	qjs "github.com/fastschema/qjs"
)

const signalSrc = `
(function() {
	function AbortSignal() {
		this.aborted = false;
		this.reason = undefined;
		this._listeners = [];
	}
	AbortSignal.prototype.addEventListener = function(type, fn) {
		if (type !== 'abort') return;
		this._listeners.push(fn);
	};
	AbortSignal.prototype.removeEventListener = function(type, fn) {
		if (type !== 'abort') return;
		this._listeners = this._listeners.filter(function(f) { return f !== fn; });
	};
	AbortSignal.prototype.throwIfAborted = function() {
		if (this.aborted) throw this.reason;
	};
	globalThis.AbortSignal = AbortSignal;
})();
`

const controllerSrc = `
(function() {
	function AbortController() {
		this.signal = new AbortSignal();
	}
	AbortController.prototype.abort = function(reason) {
		if (this.signal.aborted) return;
		this.signal.aborted = true;
		if (reason === undefined) {
			try { reason = new Error('AbortError'); } catch(e) { reason = 'AbortError'; }
		}
		this.signal.reason = reason;
		var evt = { type: 'abort', target: this.signal };
		var listeners = this.signal._listeners.slice();
		for (var i = 0; i < listeners.length; i++) {
			try { listeners[i](evt); } catch(e) {}
		}
	};
	globalThis.AbortController = AbortController;
})();
`

// Enable installs AbortSignal and AbortController as globals into ctx.
func Enable(ctx *qjs.Context) error {
	v1, err := ctx.Eval("abortsignal.js", qjs.Code(signalSrc))
	if v1 != nil {
		v1.Free()
	}
	if err != nil {
		return fmt.Errorf("abortcontroller: abortsignal: %w", err)
	}

	v2, err := ctx.Eval("abortcontroller.js", qjs.Code(controllerSrc))
	if v2 != nil {
		v2.Free()
	}
	if err != nil {
		return fmt.Errorf("abortcontroller: %w", err)
	}
	return nil
}

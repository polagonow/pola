// Package abortcontroller provides AbortController and AbortSignal for the
// QuickJS VM.
//
// react-server-dom-webpack/server.browser uses AbortController for request
// cancellation. This covers:
//   - new AbortController() with .signal
//   - ac.abort(reason?) — sets signal.aborted, fires listeners synchronously
//   - signal.addEventListener / removeEventListener / throwIfAborted
package abortcontroller

import (
	"fmt"

	quickjs "github.com/buke/quickjs-go"
)

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
func Enable(ctx *quickjs.Context) error {
	ret := ctx.Eval(signalSrc, quickjs.EvalFileName("abortsignal.js"))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("abortcontroller: abortsignal: %w", ctx.Exception())
	}

	ret2 := ctx.Eval(controllerSrc, quickjs.EvalFileName("abortcontroller.js"))
	defer ret2.Free()
	if ret2.IsException() {
		return fmt.Errorf("abortcontroller: %w", ctx.Exception())
	}
	return nil
}

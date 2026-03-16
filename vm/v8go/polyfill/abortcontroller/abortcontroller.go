// Package abortcontroller provides AbortController and AbortSignal for the
// V8 VM.
//
// react-server-dom-webpack/server.browser uses AbortController for request
// cancellation. This covers:
//   - new AbortController() with .signal
//   - ac.abort(reason?) — sets signal.aborted, fires listeners synchronously
//   - signal.addEventListener / removeEventListener / throwIfAborted
package abortcontroller

import (
	"fmt"

	v8 "rogchap.com/v8go"
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
func Enable(ctx *v8.Context) error {
	if _, err := ctx.RunScript(signalSrc, "abortsignal.js"); err != nil {
		return fmt.Errorf("abortcontroller: abortsignal: %w", err)
	}
	if _, err := ctx.RunScript(controllerSrc, "abortcontroller.js"); err != nil {
		return fmt.Errorf("abortcontroller: %w", err)
	}
	return nil
}

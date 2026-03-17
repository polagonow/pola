// Package abortcontroller provides AbortController and AbortSignal for the
// modernc.org/quickjs VM.
package abortcontroller

import (
	"fmt"

	mquickjs "modernc.org/quickjs"
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

// Enable installs AbortSignal and AbortController as globals into vm.
func Enable(vm *mquickjs.VM) error {
	if _, err := vm.Eval(signalSrc, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("abortcontroller: abortsignal: %w", err)
	}
	if _, err := vm.Eval(controllerSrc, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("abortcontroller: %w", err)
	}
	return nil
}

// Package microtask sets up the custom microtask queue used by React's Flight
// encoder inside the modernc.org/quickjs VM.
//
// Globals registered:
//   - __microtaskQueue__  — a JS Array (JS code can also call .push())
//   - queueMicrotask(fn) — appends fn to the queue
//   - __drainMicrotasks__() — splices and calls all queued fns; called
//     explicitly inside __pullStream__ to force Flight work to complete
//     before chunks are collected in the same tick.
package microtask

import (
	"fmt"

	mquickjs "modernc.org/quickjs"
)

const src = `
(function() {
	var queue = [];
	globalThis.__microtaskQueue__ = queue;

	globalThis.queueMicrotask = function(fn) {
		queue.push(fn);
	};

	globalThis.__drainMicrotasks__ = function() {
		var safety = 0;
		while (queue.length > 0 && safety < 5000) {
			safety++;
			var batch = queue.splice(0);
			for (var i = 0; i < batch.length; i++) {
				try { batch[i](); } catch(e) {}
			}
		}
	};
})();
`

// Enable installs the microtask queue globals into vm.
// Must be called before messagechannel and readablestream.
func Enable(vm *mquickjs.VM) error {
	if _, err := vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		return fmt.Errorf("microtask: %w", err)
	}
	return nil
}

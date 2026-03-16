// Package microtask sets up the custom microtask queue used by React's Flight
// encoder inside the V8 VM.
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

	v8 "rogchap.com/v8go"
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

// Enable installs the microtask queue globals into ctx.
// Must be called before messagechannel and readablestream.
func Enable(ctx *v8.Context) error {
	if _, err := ctx.RunScript(src, "microtask.js"); err != nil {
		return fmt.Errorf("microtask: %w", err)
	}
	return nil
}

// Package microtask sets up the custom microtask queue used by React's Flight
// encoder inside the QuickJS VM.
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

	quickjs "github.com/buke/quickjs-go"
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
func Enable(ctx *quickjs.Context) error {
	ret := ctx.Eval(src, quickjs.EvalFileName("microtask.js"))
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("microtask: %w", ctx.Exception())
	}
	return nil
}

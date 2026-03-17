// Globals: __microtaskQueue__, queueMicrotask(fn), __drainMicrotasks__()
// Must be loaded before messagechannel and readablestream.
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

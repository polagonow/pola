// Package enginesuite contains engine-specific e2e test suites.
package enginesuite

import (
	"testing"

	"github.com/polagonow/pola/test/fixture"
)

// RunPolyfillTests verifies that the Web API polyfills are correctly injected
// into every registered JS engine and behave identically across them. Each
// sub-test runs against every VM via fixture.ForEachVM; the polyfill fixture's
// Eval returns an error when the evaluated JS throws, so each assertion is
// written to throw on failure.
func RunPolyfillTests(t *testing.T) {
	t.Helper()

	// polyEval runs src on every registered VM and fails the VM's sub-test if
	// the script throws. Each snippet is wrapped in an IIFE so top-level
	// declarations never leak between VMs sharing a global scope.
	polyEval := func(t *testing.T, name, src string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
				if err := f.Eval("(function(){\n" + src + "\n})();"); err != nil {
					t.Errorf("%s: %v", name, err)
				}
			})
		})
	}

	// Every polyfilled global (and the internal drain/pull helpers the suites
	// rely on) must be present after Enable.
	polyEval(t, "GlobalsInstalled", `
		var names = ['TextEncoder','TextDecoder','MessageChannel','ReadableStream',
			'AbortController','AbortSignal','queueMicrotask','Promise',
			'__webpack_require__','__drainMicrotasks__','__pullStream__'];
		for (var i = 0; i < names.length; i++) {
			if (typeof globalThis[names[i]] === 'undefined') {
				throw new Error('missing global: ' + names[i]);
			}
		}
	`)

	// TextEncoder/TextDecoder round-trip ASCII and encode multi-byte UTF-8.
	polyEval(t, "TextEncoding", `
		var enc = new TextEncoder();
		var dec = new TextDecoder();
		var bytes = enc.encode('AB');
		if (bytes[0] !== 65 || bytes[1] !== 66) {
			throw new Error('ascii bytes wrong: ' + bytes[0] + ',' + bytes[1]);
		}
		if (dec.decode(enc.encode('Hello, world')) !== 'Hello, world') {
			throw new Error('ascii round-trip failed');
		}
		// '€' is U+20AC → 3 UTF-8 bytes (0xE2 0x82 0xAC).
		var euro = enc.encode('€');
		if (euro.length !== 3 || euro[0] !== 0xE2 || euro[1] !== 0x82 || euro[2] !== 0xAC) {
			throw new Error('multi-byte encoding wrong: len=' + euro.length);
		}
	`)

	// queueMicrotask defers work; __drainMicrotasks__ runs it after sync code.
	polyEval(t, "MicrotaskOrdering", `
		var order = [];
		queueMicrotask(function () { order.push('micro'); });
		order.push('sync');
		if (order.join(',') !== 'sync') {
			throw new Error('microtask ran too early: ' + order.join(','));
		}
		__drainMicrotasks__();
		if (order.join(',') !== 'sync,micro') {
			throw new Error('unexpected order: ' + order.join(','));
		}
	`)

	// MessageChannel delivers port1 → port2 messages via the microtask queue.
	polyEval(t, "MessageChannelDelivery", `
		var mc = new MessageChannel();
		if (!mc.port1 || !mc.port2 || mc.port1 === mc.port2) {
			throw new Error('ports missing or identical');
		}
		var received = null;
		mc.port2.onmessage = function (e) { received = e.data; };
		mc.port1.postMessage('ping');
		if (received !== null) throw new Error('delivered synchronously');
		__drainMicrotasks__();
		if (received !== 'ping') throw new Error('not delivered: ' + received);
	`)

	// ReadableStream enqueues chunks that __pullStream__ drains in order.
	polyEval(t, "ReadableStreamPull", `
		var s = new ReadableStream({
			start: function (c) { c.enqueue('a'); c.enqueue('b'); c.close(); }
		});
		var res = __pullStream__(s);
		if (res.chunks.length !== 2 || res.chunks[0] !== 'a' || res.chunks[1] !== 'b') {
			throw new Error('unexpected chunks: ' + JSON.stringify(res.chunks));
		}
		// __pullStream__ reports done only once the buffered chunks are drained
		// and the stream is closed, so a second pull yields no chunks and done.
		var res2 = __pullStream__(s);
		if (res2.chunks.length !== 0 || !res2.done) {
			throw new Error('stream should be drained and done: ' + JSON.stringify(res2));
		}
	`)

	// AbortController flips its signal and notifies listeners on abort().
	polyEval(t, "AbortController", `
		var ac = new AbortController();
		if (ac.signal.aborted) throw new Error('should start un-aborted');
		var fired = false;
		ac.signal.addEventListener('abort', function () { fired = true; });
		ac.abort();
		if (!ac.signal.aborted) throw new Error('signal should be aborted');
		if (!fired) throw new Error('abort listener did not fire');
		var threw = false;
		try { ac.signal.throwIfAborted(); } catch (e) { threw = true; }
		if (!threw) throw new Error('throwIfAborted should throw once aborted');
	`)
}

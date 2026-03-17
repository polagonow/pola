package polyfill_test

// 07_promise.js replaces the native Promise with a pure-JS implementation
// routed through queueMicrotask so __drainMicrotasks__() controls when
// continuations fire. These tests verify that behaviour across all VMs.

import (
	"testing"

	polyfilltest "gojsx/test/polyfill"
)

func TestPromiseResolveThen(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var result = null;
			Promise.resolve(42).then(function(v) { result = v; });
			if (result !== null) throw new Error("callback fired before drain");
			__drainMicrotasks__();
			if (result !== 42) throw new Error("expected 42, got " + result);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseRejectCatch(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var reason = null;
			Promise.reject("oops")["catch"](function(r) { reason = r; });
			if (reason !== null) throw new Error("catch fired before drain");
			__drainMicrotasks__();
			if (reason !== "oops") throw new Error("expected 'oops', got " + reason);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseChaining(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var out = [];
			Promise.resolve(1)
				.then(function(v) { out.push(v); return v + 1; })
				.then(function(v) { out.push(v); return v + 1; })
				.then(function(v) { out.push(v); });
			// Three chained .then calls, each needs its own drain cycle.
			__drainMicrotasks__();
			__drainMicrotasks__();
			__drainMicrotasks__();
			if (out[0] !== 1 || out[1] !== 2 || out[2] !== 3) {
				throw new Error("chaining failed: " + JSON.stringify(out));
			}
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseAll(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var results = null;
			Promise.all([Promise.resolve(1), Promise.resolve(2), Promise.resolve(3)])
				.then(function(v) { results = v; });
			__drainMicrotasks__();
			__drainMicrotasks__();
			__drainMicrotasks__();
			if (!results || results[0] !== 1 || results[1] !== 2 || results[2] !== 3) {
				throw new Error("Promise.all failed: " + JSON.stringify(results));
			}
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseAllSettled(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var results = null;
			Promise.allSettled([Promise.resolve("ok"), Promise.reject("fail")])
				.then(function(v) { results = v; });
			__drainMicrotasks__();
			__drainMicrotasks__();
			__drainMicrotasks__();
			if (!results) throw new Error("allSettled did not resolve");
			if (results[0].status !== "fulfilled" || results[0].value !== "ok") {
				throw new Error("wrong fulfilled: " + JSON.stringify(results[0]));
			}
			if (results[1].status !== "rejected" || results[1].reason !== "fail") {
				throw new Error("wrong rejected: " + JSON.stringify(results[1]));
			}
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseRace(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var winner = null;
			Promise.race([Promise.resolve("first"), Promise.resolve("second")])
				.then(function(v) { winner = v; });
			__drainMicrotasks__();
			__drainMicrotasks__();
			if (winner !== "first") throw new Error("expected 'first', got " + winner);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseAny(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var result = null;
			Promise.any([Promise.reject("no"), Promise.resolve("yes")])
				.then(function(v) { result = v; });
			__drainMicrotasks__();
			__drainMicrotasks__();
			__drainMicrotasks__();
			if (result !== "yes") throw new Error("expected 'yes', got " + result);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseFinally(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ran = false;
			Promise.resolve("x")["finally"](function() { ran = true; });
			__drainMicrotasks__();
			__drainMicrotasks__();
			__drainMicrotasks__();
			if (!ran) throw new Error("finally did not run");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPromiseConstructorThrows(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var caught = null;
			new Promise(function() { throw new Error("executor boom"); })
				["catch"](function(e) { caught = e.message; });
			__drainMicrotasks__();
			if (caught !== "executor boom") throw new Error("expected error, got: " + caught);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

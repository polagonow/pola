package polyfill_test

import (
	"testing"

	"gojsx/test/fixture"
	_ "gojsx/test/vm"
)

func TestQueueMicrotask(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var order = [];
			queueMicrotask(function() { order.push(1); });
			queueMicrotask(function() { order.push(2); });
			__drainMicrotasks__();
			if (order[0] !== 1 || order[1] !== 2) throw new Error("wrong order: " + JSON.stringify(order));
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDrainMicrotasksEmpty(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`__drainMicrotasks__();`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDrainMicrotasksSafety(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var count = 0;
			for (var i = 0; i < 10000; i++) {
				queueMicrotask(function() { count++; });
			}
			__drainMicrotasks__();
			if (count !== 10000) throw new Error("expected 10000, got " + count);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDrainMicrotasksSwallowsErrors(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var ran = false;
			queueMicrotask(function() { throw new Error("boom"); });
			queueMicrotask(function() { ran = true; });
			__drainMicrotasks__();
			if (!ran) throw new Error("second task did not run");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMicrotaskQueueIsArray(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var len = __microtaskQueue__.length;
			__microtaskQueue__.push(function() {});
			if (__microtaskQueue__.length !== len + 1) throw new Error("push failed");
			__drainMicrotasks__();
		`); err != nil {
			t.Fatal(err)
		}
	})
}

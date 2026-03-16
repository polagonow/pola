package microtask_test

import (
	"testing"

	sobeklib "github.com/grafana/sobek"
	"gojsx/vm/sobek/polyfill/microtask"
)

func TestQueueMicrotask(t *testing.T) {
	rt := sobeklib.New()
	microtask.Enable(rt)

	_, err := rt.RunString(`
		var results = [];
		queueMicrotask(function() { results.push(1); });
		queueMicrotask(function() { results.push(2); });
		if (results.length !== 0) throw new Error("should not run before drain");
		__drainMicrotasks__();
		if (results.length !== 2) throw new Error("expected 2 results, got " + results.length);
		if (results[0] !== 1 || results[1] !== 2) throw new Error("wrong order: " + results);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDrainMicrotasksEmpty(t *testing.T) {
	rt := sobeklib.New()
	microtask.Enable(rt)

	_, err := rt.RunString(`__drainMicrotasks__();`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDrainMicrotasksSafety(t *testing.T) {
	rt := sobeklib.New()
	microtask.Enable(rt)

	_, err := rt.RunString(`
		var count = 0;
		for (var i = 0; i < 10000; i++) {
			queueMicrotask(function() { count++; });
		}
		__drainMicrotasks__();
		if (count !== 10000) throw new Error("expected 10000, got " + count);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDrainMicrotasksSwallowsErrors(t *testing.T) {
	rt := sobeklib.New()
	microtask.Enable(rt)

	_, err := rt.RunString(`
		var ran = false;
		queueMicrotask(function() { throw new Error("boom"); });
		queueMicrotask(function() { ran = true; });
		__drainMicrotasks__();
		if (!ran) throw new Error("second callback should still run after first throws");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMicrotaskQueueIsArray(t *testing.T) {
	rt := sobeklib.New()
	microtask.Enable(rt)

	_, err := rt.RunString(`
		if (!Array.isArray(__microtaskQueue__)) throw new Error("not an array");
		var called = false;
		__microtaskQueue__.push(function() { called = true; });
		__drainMicrotasks__();
		if (!called) throw new Error("direct push to array not drained");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

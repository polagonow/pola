package microtask_test

import (
	"testing"

	v8 "rogchap.com/v8go"

	"gojsx/vm/v8go/polyfill/microtask"
)

func newCtx(t *testing.T) *v8.Context {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return v8.NewContext(iso)
}

func TestQueueMicrotask(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var results = [];
		queueMicrotask(function() { results.push(1); });
		queueMicrotask(function() { results.push(2); });
		if (results.length !== 0) throw new Error("should not run before drain");
		__drainMicrotasks__();
		if (results.length !== 2) throw new Error("expected 2 results, got " + results.length);
		if (results[0] !== 1 || results[1] !== 2) throw new Error("wrong order: " + results);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDrainMicrotasksEmpty(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := ctx.RunScript(`__drainMicrotasks__();`, "test.js"); err != nil {
		t.Fatal(err)
	}
}

func TestDrainMicrotasksSafety(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var count = 0;
		for (var i = 0; i < 10000; i++) {
			queueMicrotask(function() { count++; });
		}
		__drainMicrotasks__();
		if (count !== 10000) throw new Error("expected 10000, got " + count);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDrainMicrotasksSwallowsErrors(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var ran = false;
		queueMicrotask(function() { throw new Error("boom"); });
		queueMicrotask(function() { ran = true; });
		__drainMicrotasks__();
		if (!ran) throw new Error("second callback should still run after first throws");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMicrotaskQueueIsArray(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		if (!Array.isArray(__microtaskQueue__)) throw new Error("not an array");
		var called = false;
		__microtaskQueue__.push(function() { called = true; });
		__drainMicrotasks__();
		if (!called) throw new Error("direct push to array not drained");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

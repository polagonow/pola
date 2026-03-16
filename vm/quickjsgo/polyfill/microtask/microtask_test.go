package microtask_test

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"gojsx/vm/quickjsgo/polyfill/microtask"
)

func newCtx(t *testing.T) *quickjs.Context {
	t.Helper()
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	t.Cleanup(func() { ctx.Close(); rt.Close() })
	return ctx
}

func eval(t *testing.T, ctx *quickjs.Context, src string) {
	t.Helper()
	ret := ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		t.Fatal(ctx.Exception())
	}
}

func TestQueueMicrotask(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var results = [];
		queueMicrotask(function() { results.push(1); });
		queueMicrotask(function() { results.push(2); });
		if (results.length !== 0) throw new Error("should not run before drain");
		__drainMicrotasks__();
		if (results.length !== 2) throw new Error("expected 2 results, got " + results.length);
		if (results[0] !== 1 || results[1] !== 2) throw new Error("wrong order: " + results);
	`)
}

func TestDrainMicrotasksEmpty(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `__drainMicrotasks__();`)
}

func TestDrainMicrotasksSafety(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var count = 0;
		for (var i = 0; i < 10000; i++) {
			queueMicrotask(function() { count++; });
		}
		__drainMicrotasks__();
		if (count !== 10000) throw new Error("expected 10000, got " + count);
	`)
}

func TestDrainMicrotasksSwallowsErrors(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var ran = false;
		queueMicrotask(function() { throw new Error("boom"); });
		queueMicrotask(function() { ran = true; });
		__drainMicrotasks__();
		if (!ran) throw new Error("second callback should still run after first throws");
	`)
}

func TestMicrotaskQueueIsArray(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		if (!Array.isArray(__microtaskQueue__)) throw new Error("not an array");
		var called = false;
		__microtaskQueue__.push(function() { called = true; });
		__drainMicrotasks__();
		if (!called) throw new Error("direct push to array not drained");
	`)
}

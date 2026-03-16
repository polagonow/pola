package abortcontroller_test

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"gojsx/vm/quickjsgo/polyfill/abortcontroller"
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

func TestAbortControllerAbortFiresListeners(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var ac = new AbortController();
		var fired = false;
		ac.signal.addEventListener("abort", function(evt) {
			fired = true;
			if (evt.type !== "abort") throw new Error("expected evt.type=abort, got: " + evt.type);
		});
		if (ac.signal.aborted) throw new Error("should not be aborted initially");
		ac.abort();
		if (!ac.signal.aborted) throw new Error("signal should be aborted");
		if (!fired) throw new Error("abort listener should have fired");
	`)
}

func TestAbortControllerAbortIdempotent(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var ac = new AbortController();
		var count = 0;
		ac.signal.addEventListener("abort", function() { count++; });
		ac.abort();
		ac.abort();
		if (count !== 1) throw new Error("expected 1 firing, got: " + count);
	`)
}

func TestAbortControllerCustomReason(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var ac = new AbortController();
		var myReason = new Error("cancelled");
		ac.abort(myReason);
		if (ac.signal.reason !== myReason) throw new Error("expected custom reason");
	`)
}

func TestAbortControllerDefaultReason(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var ac = new AbortController();
		ac.abort();
		if (ac.signal.reason === undefined) throw new Error("reason should be set after abort");
	`)
}

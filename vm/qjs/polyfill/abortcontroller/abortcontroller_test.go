package abortcontroller_test

import (
	"testing"

	qjs "github.com/fastschema/qjs"
	"gojsx/vm/qjs/polyfill/abortcontroller"
)

func newCtx(t *testing.T) *qjs.Context {
	t.Helper()
	rt, err := qjs.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rt.Close() })
	return rt.Context()
}

func eval(t *testing.T, ctx *qjs.Context, src string) {
	t.Helper()
	val, err := ctx.Eval("test.js", qjs.Code(src))
	if val != nil {
		val.Free()
	}
	if err != nil {
		t.Fatal(err)
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

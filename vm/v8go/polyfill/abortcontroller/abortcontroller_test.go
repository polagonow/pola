package abortcontroller_test

import (
	"testing"

	v8 "rogchap.com/v8go"
	"gojsx/vm/v8go/polyfill/abortcontroller"
)

func newCtx(t *testing.T) *v8.Context {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return v8.NewContext(iso)
}

func TestAbortControllerAbortFiresListeners(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
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
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerAbortIdempotent(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var ac = new AbortController();
		var count = 0;
		ac.signal.addEventListener("abort", function() { count++; });
		ac.abort();
		ac.abort();
		if (count !== 1) throw new Error("expected 1 firing, got: " + count);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerCustomReason(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var ac = new AbortController();
		var myReason = new Error("cancelled");
		ac.abort(myReason);
		if (ac.signal.reason !== myReason) throw new Error("expected custom reason");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerDefaultReason(t *testing.T) {
	ctx := newCtx(t)
	if err := abortcontroller.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var ac = new AbortController();
		ac.abort();
		if (ac.signal.reason === undefined) throw new Error("reason should be set after abort");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

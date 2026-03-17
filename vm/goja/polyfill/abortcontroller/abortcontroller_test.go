package abortcontroller_test

import (
	"testing"

	"github.com/dop251/goja"

	"gojsx/vm/goja/polyfill/abortcontroller"
)

func TestAbortControllerAbortFiresListeners(t *testing.T) {
	rt := goja.New()
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
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
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerAbortIdempotent(t *testing.T) {
	rt := goja.New()
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
		var ac = new AbortController();
		var count = 0;
		ac.signal.addEventListener("abort", function() { count++; });
		ac.abort();
		ac.abort();
		if (count !== 1) throw new Error("expected 1 firing, got: " + count);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerCustomReason(t *testing.T) {
	rt := goja.New()
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
		var ac = new AbortController();
		var myReason = new Error("cancelled");
		ac.abort(myReason);
		if (ac.signal.reason !== myReason) throw new Error("expected custom reason");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerDefaultReason(t *testing.T) {
	rt := goja.New()
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
		var ac = new AbortController();
		ac.abort();
		if (ac.signal.reason === undefined) throw new Error("reason should be set after abort");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

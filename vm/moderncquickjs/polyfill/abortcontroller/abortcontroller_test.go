package abortcontroller_test

import (
	"testing"

	mquickjs "modernc.org/quickjs"

	"gojsx/vm/moderncquickjs/polyfill/abortcontroller"
)

func newVM(t *testing.T) *mquickjs.VM {
	t.Helper()
	vm, err := mquickjs.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { vm.Close() })
	return vm
}

func eval(t *testing.T, vm *mquickjs.VM, src string) {
	t.Helper()
	if _, err := vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		t.Fatal(err)
	}
}

func TestAbortControllerAbortFiresListeners(t *testing.T) {
	vm := newVM(t)
	if err := abortcontroller.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
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
	vm := newVM(t)
	if err := abortcontroller.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		var ac = new AbortController();
		var count = 0;
		ac.signal.addEventListener("abort", function() { count++; });
		ac.abort();
		ac.abort();
		if (count !== 1) throw new Error("expected 1 firing, got: " + count);
	`)
}

func TestAbortControllerCustomReason(t *testing.T) {
	vm := newVM(t)
	if err := abortcontroller.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		var ac = new AbortController();
		var myReason = new Error("cancelled");
		ac.abort(myReason);
		if (ac.signal.reason !== myReason) throw new Error("expected custom reason");
	`)
}

func TestAbortControllerDefaultReason(t *testing.T) {
	vm := newVM(t)
	if err := abortcontroller.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		var ac = new AbortController();
		ac.abort();
		if (ac.signal.reason === undefined) throw new Error("reason should be set after abort");
	`)
}

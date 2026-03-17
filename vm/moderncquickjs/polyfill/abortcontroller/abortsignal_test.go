package abortcontroller_test

import (
	"testing"

	"gojsx/vm/moderncquickjs/polyfill/abortcontroller"
)

func TestAbortSignalRemoveEventListener(t *testing.T) {
	vm := newVM(t)
	abortcontroller.Enable(vm) //nolint:errcheck

	eval(t, vm, `
		var ac = new AbortController();
		var count = 0;
		var fn = function() { count++; };
		ac.signal.addEventListener("abort", fn);
		ac.signal.removeEventListener("abort", fn);
		ac.abort();
		if (count !== 0) throw new Error("listener should have been removed, but fired " + count + " time(s)");
	`)
}

func TestAbortSignalThrowIfAborted(t *testing.T) {
	vm := newVM(t)
	abortcontroller.Enable(vm) //nolint:errcheck

	eval(t, vm, `
		var ac = new AbortController();
		ac.signal.throwIfAborted();
		ac.abort();
		var threw = false;
		try {
			ac.signal.throwIfAborted();
		} catch (e) {
			threw = true;
		}
		if (!threw) throw new Error("throwIfAborted should throw after abort");
	`)
}

func TestAbortSignalIgnoresNonAbortEvents(t *testing.T) {
	vm := newVM(t)
	abortcontroller.Enable(vm) //nolint:errcheck

	eval(t, vm, `
		var ac = new AbortController();
		var count = 0;
		ac.signal.addEventListener("click", function() { count++; });
		ac.abort();
		if (count !== 0) throw new Error("non-abort listener should not fire");
	`)
}

func TestAbortSignalMultipleListeners(t *testing.T) {
	vm := newVM(t)
	abortcontroller.Enable(vm) //nolint:errcheck

	eval(t, vm, `
		var ac = new AbortController();
		var results = [];
		ac.signal.addEventListener("abort", function() { results.push(1); });
		ac.signal.addEventListener("abort", function() { results.push(2); });
		ac.abort();
		if (results.length !== 2) throw new Error("expected 2 listeners fired, got: " + results.length);
	`)
}

package abortcontroller_test

import (
	"testing"

	"gojsx/vm/sobek/polyfill/abortcontroller"
)

func TestAbortSignalRemoveEventListener(t *testing.T) {
	rt := newRT(t)
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
		var ac = new AbortController();
		var count = 0;
		var fn = function() { count++; };
		ac.signal.addEventListener("abort", fn);
		ac.signal.removeEventListener("abort", fn);
		ac.abort();
		if (count !== 0) throw new Error("listener should have been removed, but fired " + count + " time(s)");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortSignalThrowIfAborted(t *testing.T) {
	rt := newRT(t)
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
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
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortSignalIgnoresNonAbortEvents(t *testing.T) {
	rt := newRT(t)
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
		var ac = new AbortController();
		var count = 0;
		ac.signal.addEventListener("click", function() { count++; });
		ac.abort();
		if (count !== 0) throw new Error("non-abort listener should not fire");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAbortSignalMultipleListeners(t *testing.T) {
	rt := newRT(t)
	abortcontroller.Enable(rt)

	_, err := rt.RunString(`
		var ac = new AbortController();
		var results = [];
		ac.signal.addEventListener("abort", function() { results.push(1); });
		ac.signal.addEventListener("abort", function() { results.push(2); });
		ac.abort();
		if (results.length !== 2) throw new Error("expected 2 listeners fired, got: " + results.length);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

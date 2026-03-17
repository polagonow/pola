package polyfill_test

import (
	"testing"

	polyfilltest "gojsx/test/polyfill"
)

func TestAbortControllerAbortFiresListeners(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			var fired = null;
			ctrl.signal.addEventListener("abort", function(e) { fired = e.type; });
			ctrl.abort();
			if (fired !== "abort") throw new Error("listener not fired, got: " + fired);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortControllerAbortIdempotent(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			var count = 0;
			ctrl.signal.addEventListener("abort", function() { count++; });
			ctrl.abort();
			ctrl.abort();
			if (count !== 1) throw new Error("expected 1 fire, got " + count);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortControllerCustomReason(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			ctrl.abort("my-reason");
			if (ctrl.signal.reason !== "my-reason") throw new Error("wrong reason: " + ctrl.signal.reason);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortControllerDefaultReason(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			ctrl.abort();
			if (!ctrl.signal.reason) throw new Error("no default reason set");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortSignalRemoveEventListener(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			var removed = 0;
			var handler = function() { removed++; };
			ctrl.signal.addEventListener("abort", handler);
			ctrl.signal.removeEventListener("abort", handler);
			ctrl.abort();
			if (removed !== 0) throw new Error("removed listener fired");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortSignalThrowIfAborted(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			var threw = false;
			try { ctrl.signal.throwIfAborted(); } catch(e) { threw = true; }
			if (threw) throw new Error("should not throw before abort");
			ctrl.abort();
			var threw2 = false;
			try { ctrl.signal.throwIfAborted(); } catch(e) { threw2 = true; }
			if (!threw2) throw new Error("should throw after abort");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortSignalIgnoresNonAbortEvents(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			var clicked = false;
			ctrl.signal.addEventListener("click", function() { clicked = true; });
			ctrl.abort();
			if (clicked) throw new Error("click listener should be ignored");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAbortSignalMultipleListeners(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ctrl = new AbortController();
			var f1 = false, f2 = false;
			ctrl.signal.addEventListener("abort", function() { f1 = true; });
			ctrl.signal.addEventListener("abort", function() { f2 = true; });
			ctrl.abort();
			if (!f1 || !f2) throw new Error("not all listeners fired");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

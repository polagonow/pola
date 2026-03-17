package polyfilltest

import "testing"

func TestReadableStreamEnqueueClose(t *testing.T) {
	forEachVM(t, func(t *testing.T, f Fixture) {
		if err := f.Eval(`
			var chunks = [];
			var s = new ReadableStream({
				start: function(ctrl) {
					ctrl.enqueue("a");
					ctrl.enqueue("b");
					ctrl.enqueue("c");
					ctrl.close();
				}
			});
			var r;
			while (true) {
				r = __pullStream__(s);
				for (var i = 0; i < r.chunks.length; i++) chunks.push(r.chunks[i]);
				if (r.done) break;
			}
			if (chunks.length !== 3 || chunks[0] !== "a" || chunks[1] !== "b" || chunks[2] !== "c") {
				throw new Error("wrong chunks: " + JSON.stringify(chunks));
			}
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadableStreamErrorCloses(t *testing.T) {
	forEachVM(t, func(t *testing.T, f Fixture) {
		if err := f.Eval(`
			var s = new ReadableStream({
				start: function(ctrl) { ctrl.error(new Error("boom")); }
			});
			var r = __pullStream__(s);
			if (!r.done) throw new Error("should be done after error");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadableStreamStartCalledOnce(t *testing.T) {
	forEachVM(t, func(t *testing.T, f Fixture) {
		if err := f.Eval(`
			var calls = 0;
			var s = new ReadableStream({
				start: function(ctrl) { calls++; ctrl.close(); }
			});
			__pullStream__(s);
			__pullStream__(s);
			if (calls !== 1) throw new Error("start called " + calls + " times, expected 1");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadableStreamPullCalled(t *testing.T) {
	forEachVM(t, func(t *testing.T, f Fixture) {
		if err := f.Eval(`
			var pullCount = 0;
			var s = new ReadableStream({
				pull: function(ctrl) { pullCount++; ctrl.close(); }
			});
			__pullStream__(s);
			if (pullCount < 1) throw new Error("pull not called");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadableStreamDesiredSize(t *testing.T) {
	forEachVM(t, func(t *testing.T, f Fixture) {
		if err := f.Eval(`
			var s = new ReadableStream({});
			if (s._controller.desiredSize !== 1) throw new Error("expected desiredSize 1");
			s._controller.close();
			if (s._controller.desiredSize !== 0) throw new Error("expected desiredSize 0 after close");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadableStreamByobRequestNull(t *testing.T) {
	forEachVM(t, func(t *testing.T, f Fixture) {
		if err := f.Eval(`
			var s = new ReadableStream({});
			if (s._controller.byobRequest !== null) throw new Error("byobRequest should be null");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

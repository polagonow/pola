package readablestream_test

import (
	"testing"

	"github.com/dop251/goja"

	"gojsx/vm/goja/polyfill/microtask"
	"gojsx/vm/goja/polyfill/readablestream"
)

func TestReadableStreamEnqueueClose(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	readablestream.Enable(rt)

	_, err := rt.RunString(`
		var stream = new ReadableStream({
			start: function(controller) {
				controller.enqueue("chunk1");
				controller.enqueue("chunk2");
				controller.enqueue("chunk3");
				controller.close();
			}
		});

		var allChunks = [];
		var done = false;
		var iterations = 0;
		while (!done && iterations++ < 100) {
			var result = __pullStream__(stream);
			for (var i = 0; i < result.chunks.length; i++) {
				allChunks.push(result.chunks[i]);
			}
			done = result.done;
		}

		if (allChunks.length !== 3) throw new Error("expected 3 chunks, got " + allChunks.length);
		if (allChunks[0] !== "chunk1") throw new Error("expected chunk1, got " + allChunks[0]);
		if (allChunks[2] !== "chunk3") throw new Error("expected chunk3, got " + allChunks[2]);
		if (!done) throw new Error("stream should be done");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadableStreamErrorCloses(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	readablestream.Enable(rt)

	_, err := rt.RunString(`
		var stream = new ReadableStream({
			start: function(controller) {
				controller.enqueue("before-error");
				controller.error(new Error("stream error"));
			}
		});

		var result = __pullStream__(stream);
		if (result.chunks.length !== 1) throw new Error("expected 1 chunk before error, got " + result.chunks.length);
		var result2 = __pullStream__(stream);
		if (!result2.done) throw new Error("stream should be done after error");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadableStreamStartCalledOnce(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	readablestream.Enable(rt)

	_, err := rt.RunString(`
		var startCount = 0;
		var stream = new ReadableStream({
			start: function(controller) {
				startCount++;
				controller.close();
			}
		});

		__pullStream__(stream);
		__pullStream__(stream);
		__pullStream__(stream);

		if (startCount !== 1) throw new Error("start should be called once, called " + startCount);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadableStreamPullCalled(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	readablestream.Enable(rt)

	_, err := rt.RunString(`
		var pullCount = 0;
		var stream = new ReadableStream({
			start: function(controller) {},
			pull: function(controller) {
				pullCount++;
				if (pullCount >= 3) controller.close();
			}
		});

		var done = false;
		var iterations = 0;
		while (!done && iterations++ < 100) {
			var result = __pullStream__(stream);
			done = result.done;
		}

		if (pullCount < 3) throw new Error("pull should be called at least 3 times, called " + pullCount);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadableStreamDesiredSize(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	readablestream.Enable(rt)

	_, err := rt.RunString(`
		var stream = new ReadableStream({
			start: function(controller) {
				if (controller.desiredSize !== 1) throw new Error("expected desiredSize=1, got " + controller.desiredSize);
				controller.close();
				if (controller.desiredSize !== 0) throw new Error("expected desiredSize=0 after close, got " + controller.desiredSize); //nolint:lll
			}
		});
		__pullStream__(stream);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadableStreamByobRequestNull(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	readablestream.Enable(rt)

	_, err := rt.RunString(`
		var stream = new ReadableStream({
			start: function(controller) {
				if (controller.byobRequest !== null) throw new Error("byobRequest should be null");
				controller.close();
			}
		});
		__pullStream__(stream);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

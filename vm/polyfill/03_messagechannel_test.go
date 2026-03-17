package polyfill_test

import (
	"testing"

	polyfilltest "gojsx/test/polyfill"
)

func TestMessageChannelPostMessage(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ch = new MessageChannel();
			var received = null;
			ch.port1.onmessage = function(e) { received = e.data; };
			ch.port2.postMessage("ping");
			__drainMicrotasks__();
			if (received !== "ping") throw new Error("expected 'ping', got " + received);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMessageChannelBidirectional(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ch = new MessageChannel();
			var r1 = null, r2 = null;
			ch.port1.onmessage = function(e) { r1 = e.data; };
			ch.port2.onmessage = function(e) { r2 = e.data; };
			ch.port2.postMessage("toPort1");
			ch.port1.postMessage("toPort2");
			__drainMicrotasks__();
			if (r1 !== "toPort1" || r2 !== "toPort2") throw new Error("bidirectional failed: " + r1 + " " + r2);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMessageChannelNoHandlerNoError(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ch = new MessageChannel();
			ch.port2.postMessage("no handler");
			__drainMicrotasks__();
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMessageChannelObjectData(t *testing.T) {
	polyfilltest.ForEachVM(t, func(t *testing.T, f polyfilltest.Fixture) {
		if err := f.Eval(`
			var ch = new MessageChannel();
			var obj = null;
			ch.port1.onmessage = function(e) { obj = e.data; };
			ch.port2.postMessage({value: 42});
			__drainMicrotasks__();
			if (!obj || obj.value !== 42) throw new Error("object data failed: " + JSON.stringify(obj));
		`); err != nil {
			t.Fatal(err)
		}
	})
}

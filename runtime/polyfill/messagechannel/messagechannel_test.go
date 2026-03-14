package messagechannel_test

import (
	"testing"

	"github.com/dop251/goja"
	"gojsx/runtime/polyfill/messagechannel"
	"gojsx/runtime/polyfill/microtask"
)

func TestMessageChannelPostMessage(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	messagechannel.Enable(rt)

	_, err := rt.RunString(`
		var mc = new MessageChannel();
		var received = null;
		mc.port1.onmessage = function(evt) { received = evt.data; };
		mc.port2.postMessage("hello");
		if (received !== null) throw new Error("should not deliver before drain");
		__drainMicrotasks__();
		if (received !== "hello") throw new Error("expected 'hello', got: " + received);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMessageChannelBidirectional(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	messagechannel.Enable(rt)

	_, err := rt.RunString(`
		var mc = new MessageChannel();
		var from1 = null, from2 = null;
		mc.port1.onmessage = function(evt) { from2 = evt.data; };
		mc.port2.onmessage = function(evt) { from1 = evt.data; };
		mc.port1.postMessage("from-port1");
		mc.port2.postMessage("from-port2");
		__drainMicrotasks__();
		if (from1 !== "from-port1") throw new Error("port2 expected 'from-port1', got: " + from1);
		if (from2 !== "from-port2") throw new Error("port1 expected 'from-port2', got: " + from2);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMessageChannelNoHandlerNoError(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	messagechannel.Enable(rt)

	_, err := rt.RunString(`
		var mc = new MessageChannel();
		mc.port2.postMessage("ignored");
		__drainMicrotasks__();
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMessageChannelObjectData(t *testing.T) {
	rt := goja.New()
	microtask.Enable(rt)
	messagechannel.Enable(rt)

	_, err := rt.RunString(`
		var mc = new MessageChannel();
		var received = null;
		mc.port1.onmessage = function(evt) { received = evt.data; };
		mc.port2.postMessage({ value: 42 });
		__drainMicrotasks__();
		if (!received || received.value !== 42) throw new Error("expected {value:42}, got: " + JSON.stringify(received));
	`)
	if err != nil {
		t.Fatal(err)
	}
}

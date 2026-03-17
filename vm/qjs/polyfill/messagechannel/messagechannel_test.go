package messagechannel_test

import (
	"testing"

	qjs "github.com/fastschema/qjs"
	"gojsx/vm/qjs/polyfill/messagechannel"
	"gojsx/vm/qjs/polyfill/microtask"
)

func newCtx(t *testing.T) *qjs.Context {
	t.Helper()
	rt, err := qjs.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rt.Close() })
	return rt.Context()
}

func eval(t *testing.T, ctx *qjs.Context, src string) {
	t.Helper()
	val, err := ctx.Eval("test.js", qjs.Code(src))
	if val != nil {
		val.Free()
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestMessageChannelPostMessage(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	if err := messagechannel.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var mc = new MessageChannel();
		var received = null;
		mc.port1.onmessage = function(evt) { received = evt.data; };
		mc.port2.postMessage("hello");
		if (received !== null) throw new Error("should not deliver before drain");
		__drainMicrotasks__();
		if (received !== "hello") throw new Error("expected 'hello', got: " + received);
	`)
}

func TestMessageChannelBidirectional(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	if err := messagechannel.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
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
}

func TestMessageChannelNoHandlerNoError(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	if err := messagechannel.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var mc = new MessageChannel();
		mc.port2.postMessage("ignored");
		__drainMicrotasks__();
	`)
}

func TestMessageChannelObjectData(t *testing.T) {
	ctx := newCtx(t)
	if err := microtask.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	if err := messagechannel.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var mc = new MessageChannel();
		var received = null;
		mc.port1.onmessage = function(evt) { received = evt.data; };
		mc.port2.postMessage({ value: 42 });
		__drainMicrotasks__();
		if (!received || received.value !== 42) throw new Error("expected {value:42}, got: " + JSON.stringify(received));
	`)
}

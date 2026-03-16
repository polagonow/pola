package messagechannel_test

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"gojsx/vm/quickjsgo/polyfill/messagechannel"
	"gojsx/vm/quickjsgo/polyfill/microtask"
)

func newCtx(t *testing.T) *quickjs.Context {
	t.Helper()
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	t.Cleanup(func() { ctx.Close(); rt.Close() })
	return ctx
}

func eval(t *testing.T, ctx *quickjs.Context, src string) {
	t.Helper()
	ret := ctx.Eval(src, quickjs.EvalFileName("test.js"))
	defer ret.Free()
	if ret.IsException() {
		t.Fatal(ctx.Exception())
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

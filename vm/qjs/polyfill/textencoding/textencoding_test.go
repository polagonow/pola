package textencoding_test

import (
	"testing"

	qjs "github.com/fastschema/qjs"
	"gojsx/vm/qjs/polyfill/textencoding"
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

func TestTextEncoderASCII(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var enc = new TextEncoder();
		var bytes = enc.encode("hello");
		if (bytes.length !== 5) throw new Error("expected 5 bytes, got " + bytes.length);
		if (bytes[0] !== 104) throw new Error("expected 104 ('h'), got " + bytes[0]);
	`)
}

func TestTextEncoderUnicode(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var enc = new TextEncoder();
		var bytes = enc.encode("\u00e9"); // é = 0xC3 0xA9
		if (bytes.length !== 2) throw new Error("expected 2 bytes for é, got " + bytes.length);
		if (bytes[0] !== 0xC3 || bytes[1] !== 0xA9) throw new Error("wrong UTF-8 bytes for é");
	`)
}

func TestTextDecoderASCII(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var dec = new TextDecoder();
		var str = dec.decode(new Uint8Array([104, 101, 108, 108, 111]));
		if (str !== "hello") throw new Error("expected 'hello', got: " + str);
	`)
}

func TestTextRoundtrip(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var enc = new TextEncoder();
		var dec = new TextDecoder();
		var original = "Hello, \u4e16\u754c!"; // Hello, 世界!
		var encoded = enc.encode(original);
		var decoded = dec.decode(encoded);
		if (decoded !== original) throw new Error("roundtrip failed: " + decoded);
	`)
}

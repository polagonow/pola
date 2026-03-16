package textencoding_test

import (
	"testing"

	v8 "rogchap.com/v8go"
	"gojsx/vm/v8go/polyfill/textencoding"
)

func newCtx(t *testing.T) *v8.Context {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return v8.NewContext(iso)
}

func TestTextEncoderEncodeASCII(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var enc = new TextEncoder();
		var buf = enc.encode("hello");
		if (!(buf instanceof Uint8Array)) throw new Error("not Uint8Array");
		if (buf.length !== 5) throw new Error("expected length 5, got " + buf.length);
		if (buf[0] !== 104) throw new Error("expected 104 ('h'), got " + buf[0]);
		if (buf[4] !== 111) throw new Error("expected 111 ('o'), got " + buf[4]);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTextEncoderEncodeUTF8MultiByte(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	// € is U+20AC → UTF-8: 0xE2 0x82 0xAC
	_, err := ctx.RunScript(`
		var enc = new TextEncoder();
		var buf = enc.encode("€");
		if (buf.length !== 3) throw new Error("expected 3 bytes, got " + buf.length);
		if (buf[0] !== 0xE2) throw new Error("byte 0: expected 0xE2, got " + buf[0].toString(16));
		if (buf[1] !== 0x82) throw new Error("byte 1: expected 0x82, got " + buf[1].toString(16));
		if (buf[2] !== 0xAC) throw new Error("byte 2: expected 0xAC, got " + buf[2].toString(16));
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTextEncoderEncoding(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var enc = new TextEncoder();
		if (enc.encoding !== "utf-8") throw new Error("expected utf-8, got " + enc.encoding);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTextDecoderDecodeBytes(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var dec = new TextDecoder();
		var buf = new Uint8Array([104, 101, 108, 108, 111]); // "hello"
		var s = dec.decode(buf);
		if (s !== "hello") throw new Error("expected 'hello', got '" + s + "'");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTextDecoderDecodeUndefined(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var dec = new TextDecoder();
		if (dec.decode(undefined) !== "") throw new Error("undefined should return empty string");
		if (dec.decode(null) !== "") throw new Error("null should return empty string");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTextEncodeDecodeRoundtrip(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var enc = new TextEncoder();
		var dec = new TextDecoder();
		var original = "Hello, 世界! €";
		var encoded = enc.encode(original);
		var decoded = dec.decode(encoded);
		if (decoded !== original) throw new Error("roundtrip failed: '" + decoded + "' !== '" + original + "'");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTextEncoderEncodeInto(t *testing.T) {
	ctx := newCtx(t)
	if err := textencoding.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var enc = new TextEncoder();
		var dest = new Uint8Array(10);
		var result = enc.encodeInto("hi", dest);
		if (result.written !== 2) throw new Error("expected written=2, got " + result.written);
		if (dest[0] !== 104) throw new Error("expected 104 ('h'), got " + dest[0]);
		if (dest[1] !== 105) throw new Error("expected 105 ('i'), got " + dest[1]);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

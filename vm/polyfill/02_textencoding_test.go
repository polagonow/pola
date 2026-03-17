package polyfill_test

import (
	"testing"

	"gojsx/test/fixture"
	_ "gojsx/test/vm"
)

func TestTextEncoderEncodeASCII(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var enc = new TextEncoder();
			var buf = enc.encode("hello");
			if (buf.length !== 5) throw new Error("expected 5 bytes, got " + buf.length);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTextEncoderEncodeUTF8MultiByte(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var enc = new TextEncoder();
			var buf = enc.encode("€");
			if (buf.length !== 3) throw new Error("expected 3 bytes for €, got " + buf.length);
			if (buf[0] !== 0xE2 || buf[1] !== 0x82 || buf[2] !== 0xAC) throw new Error("wrong bytes: " + [buf[0],buf[1],buf[2]]);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTextEncoderEncoding(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var enc = new TextEncoder();
			if (enc.encoding !== "utf-8") throw new Error("expected utf-8, got " + enc.encoding);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTextDecoderDecodeBytes(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var dec = new TextDecoder();
			var bytes = new Uint8Array([104,101,108,108,111]);
			var result = dec.decode(bytes);
			if (result !== "hello") throw new Error("expected 'hello', got " + result);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTextDecoderDecodeUndefined(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var dec = new TextDecoder();
			if (dec.decode(undefined) !== "") throw new Error("undefined should decode to empty");
			if (dec.decode(null) !== "") throw new Error("null should decode to empty");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTextEncodeDecodeRoundtrip(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var enc = new TextEncoder();
			var dec = new TextDecoder();
			var original = "Hello, \u4e16\u754c! \u20ac";
			var encoded = enc.encode(original);
			var decoded = dec.decode(encoded);
			if (decoded !== original) throw new Error("roundtrip failed: " + decoded);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTextEncoderEncodeInto(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var enc = new TextEncoder();
			var dest = new Uint8Array(10);
			enc.encodeInto("hello", dest);
			if (dest[0] !== 104) throw new Error("expected 104, got " + dest[0]);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

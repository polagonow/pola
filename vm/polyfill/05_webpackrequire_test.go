package polyfill_test

import (
	"testing"

	"gojsx/test/fixture"
	_ "gojsx/test/vm"
)

func TestWebpackRequireUnknownId(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var result = __webpack_require__("unknown");
			if (result.status !== "fulfilled" || result.value !== null) {
				throw new Error("unexpected: " + JSON.stringify(result));
			}
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWebpackRequireRegisteredId(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			__webpackModuleRegistry__["mymod"] = {exports: "hello"};
			var mod = __webpack_require__("mymod");
			if (mod.exports !== "hello") throw new Error("registered module not returned: " + JSON.stringify(mod));
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWebpackRequireU(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var url = __webpack_require__.u("chunk42");
			if (url !== "chunk42") throw new Error("expected 'chunk42', got " + url);
		`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWebpackChunkLoad(t *testing.T) {
	fixture.ForEachVM(t, func(t *testing.T, f fixture.PolyfillFixture) {
		if err := f.Eval(`
			var p = __webpack_chunk_load__("chunk1");
			if (!p || typeof p.then !== "function") throw new Error("expected Promise-like");
		`); err != nil {
			t.Fatal(err)
		}
	})
}

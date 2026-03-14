package webpackrequire_test

import (
	"testing"

	"github.com/dop251/goja"
	"gojsx/runtime/polyfill/webpackrequire"
)

func TestWebpackRequireUnknownId(t *testing.T) {
	rt := goja.New()
	webpackrequire.Enable(rt)

	_, err := rt.RunString(`
		var result = __webpack_require__("unknown-module");
		if (result.status !== "fulfilled") throw new Error("expected status=fulfilled, got: " + result.status);
		if (result.value !== null) throw new Error("expected value=null, got: " + result.value);
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebpackRequireRegisteredId(t *testing.T) {
	rt := goja.New()
	webpackrequire.Enable(rt)

	_, err := rt.RunString(`
		__webpackModuleRegistry__["my-module"] = { default: function() {} };
		var result = __webpack_require__("my-module");
		if (typeof result.default !== "function") throw new Error("expected module with default export");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebpackRequireU(t *testing.T) {
	rt := goja.New()
	webpackrequire.Enable(rt)

	_, err := rt.RunString(`
		var id = "some-chunk-id";
		if (__webpack_require__.u(id) !== id) throw new Error("u() should return its argument");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebpackChunkLoad(t *testing.T) {
	rt := goja.New()
	webpackrequire.Enable(rt)

	_, err := rt.RunString(`
		var p = __webpack_chunk_load__("chunk-1");
		if (typeof p !== "object" || p === null) throw new Error("expected a Promise-like object");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

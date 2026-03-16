package webpackrequire_test

import (
	"testing"

	v8 "rogchap.com/v8go"
	"gojsx/vm/v8go/polyfill/webpackrequire"
)

func newCtx(t *testing.T) *v8.Context {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return v8.NewContext(iso)
}

func TestWebpackRequireUnknownId(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var result = __webpack_require__("unknown-module");
		if (result.status !== "fulfilled") throw new Error("expected status=fulfilled, got: " + result.status);
		if (result.value !== null) throw new Error("expected value=null, got: " + result.value);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebpackRequireRegisteredId(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		__webpackModuleRegistry__["my-module"] = { default: function() {} };
		var result = __webpack_require__("my-module");
		if (typeof result.default !== "function") throw new Error("expected module with default export");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebpackRequireU(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var id = "some-chunk-id";
		if (__webpack_require__.u(id) !== id) throw new Error("u() should return its argument");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWebpackChunkLoad(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		var p = __webpack_chunk_load__("chunk-1");
		if (typeof p !== "object" || p === null) throw new Error("expected a Promise-like object");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

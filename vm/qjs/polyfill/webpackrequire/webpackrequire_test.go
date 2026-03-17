package webpackrequire_test

import (
	"testing"

	qjs "github.com/fastschema/qjs"
	"gojsx/vm/qjs/polyfill/webpackrequire"
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

func TestWebpackRequireUnknownId(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var result = __webpack_require__("unknown-module");
		if (result.status !== "fulfilled") throw new Error("expected status=fulfilled, got: " + result.status);
		if (result.value !== null) throw new Error("expected value=null, got: " + result.value);
	`)
}

func TestWebpackRequireRegisteredId(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		__webpackModuleRegistry__["my-module"] = { default: function() {} };
		var result = __webpack_require__("my-module");
		if (typeof result.default !== "function") throw new Error("expected module with default export");
	`)
}

func TestWebpackRequireU(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var id = "some-chunk-id";
		if (__webpack_require__.u(id) !== id) throw new Error("u() should return its argument");
	`)
}

func TestWebpackChunkLoad(t *testing.T) {
	ctx := newCtx(t)
	if err := webpackrequire.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	eval(t, ctx, `
		var p = __webpack_chunk_load__("chunk-1");
		if (typeof p !== "object" || p === null) throw new Error("expected a Promise-like object");
	`)
}

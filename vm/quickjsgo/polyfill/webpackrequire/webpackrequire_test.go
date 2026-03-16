package webpackrequire_test

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
	"gojsx/vm/quickjsgo/polyfill/webpackrequire"
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

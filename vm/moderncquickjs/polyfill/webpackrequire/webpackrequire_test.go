package webpackrequire_test

import (
	"testing"

	mquickjs "modernc.org/quickjs"

	"gojsx/vm/moderncquickjs/polyfill/webpackrequire"
)

func newVM(t *testing.T) *mquickjs.VM {
	t.Helper()
	vm, err := mquickjs.NewVM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { vm.Close() })
	return vm
}

func eval(t *testing.T, vm *mquickjs.VM, src string) {
	t.Helper()
	if _, err := vm.Eval(src, mquickjs.EvalGlobal); err != nil {
		t.Fatal(err)
	}
}

func TestWebpackRequireUnknownId(t *testing.T) {
	vm := newVM(t)
	if err := webpackrequire.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		var result = __webpack_require__("unknown-module");
		if (result.status !== "fulfilled") throw new Error("expected status=fulfilled, got: " + result.status);
		if (result.value !== null) throw new Error("expected value=null, got: " + result.value);
	`)
}

func TestWebpackRequireRegisteredId(t *testing.T) {
	vm := newVM(t)
	if err := webpackrequire.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		__webpackModuleRegistry__["my-module"] = { default: function() {} };
		var result = __webpack_require__("my-module");
		if (typeof result.default !== "function") throw new Error("expected module with default export");
	`)
}

func TestWebpackRequireU(t *testing.T) {
	vm := newVM(t)
	if err := webpackrequire.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		var id = "some-chunk-id";
		if (__webpack_require__.u(id) !== id) throw new Error("u() should return its argument");
	`)
}

func TestWebpackChunkLoad(t *testing.T) {
	vm := newVM(t)
	if err := webpackrequire.Enable(vm); err != nil {
		t.Fatal(err)
	}
	eval(t, vm, `
		var p = __webpack_chunk_load__("chunk-1");
		if (typeof p !== "object" || p === null) throw new Error("expected a Promise-like object");
	`)
}

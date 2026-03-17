package jsi_test

import (
	"testing"

	v8 "rogchap.com/v8go"

	"gojsx/vm/v8go/polyfill/jsi"
)

func newCtx(t *testing.T) *v8.Context {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return v8.NewContext(iso)
}

func TestJSIIsObject(t *testing.T) {
	ctx := newCtx(t)
	if err := jsi.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		if (typeof __JSI__ !== "object" || __JSI__ === null) throw new Error("__JSI__ should be a non-null object");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestJSIInitiallyEmpty(t *testing.T) {
	ctx := newCtx(t)
	if err := jsi.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		if (Object.keys(__JSI__).length !== 0) throw new Error("__JSI__ should be empty initially");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestJSIAcceptsProperties(t *testing.T) {
	ctx := newCtx(t)
	if err := jsi.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		__JSI__.myFn = function() { return 42; };
		if (typeof __JSI__.myFn !== "function") throw new Error("should be able to set properties on __JSI__");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

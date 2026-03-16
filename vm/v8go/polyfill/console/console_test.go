package console_test

import (
	"testing"

	v8 "rogchap.com/v8go"
	"gojsx/vm/v8go/polyfill/console"
)

func newCtx(t *testing.T) (*v8.Isolate, *v8.Context) {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return iso, v8.NewContext(iso)
}

func TestConsoleMethods(t *testing.T) {
	iso, ctx := newCtx(t)
	if err := console.Enable(iso, ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		if (typeof console !== "object") throw new Error("console should be an object");
		if (typeof console.log !== "function") throw new Error("console.log should be a function");
		if (typeof console.warn !== "function") throw new Error("console.warn should be a function");
		if (typeof console.error !== "function") throw new Error("console.error should be a function");
		if (typeof console.info !== "function") throw new Error("console.info should be a function");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsoleLogDoesNotThrow(t *testing.T) {
	iso, ctx := newCtx(t)
	if err := console.Enable(iso, ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		console.log("hello", "world");
		console.warn("a warning");
		console.error("an error");
		console.info("info message");
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConsoleLogMultipleArgs(t *testing.T) {
	iso, ctx := newCtx(t)
	if err := console.Enable(iso, ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		console.log(1, 2, 3, "four");
		console.log({ key: "value" });
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

package globals_test

import (
	"testing"

	v8 "rogchap.com/v8go"
	"gojsx/vm/v8go/polyfill/globals"
)

func newCtx(t *testing.T) *v8.Context {
	t.Helper()
	iso := v8.NewIsolate()
	t.Cleanup(iso.Dispose)
	return v8.NewContext(iso)
}

func TestProcessEnv(t *testing.T) {
	ctx := newCtx(t)
	if err := globals.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		if (typeof process !== "object") throw new Error("process should be an object");
		if (process.env.NODE_ENV !== "production") throw new Error("expected NODE_ENV=production, got: " + process.env.NODE_ENV);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPerformanceNow(t *testing.T) {
	ctx := newCtx(t)
	if err := globals.Enable(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := ctx.RunScript(`
		if (typeof performance !== "object") throw new Error("performance should be an object");
		if (typeof performance.now !== "function") throw new Error("performance.now should be a function");
		var t = performance.now();
		if (typeof t !== "number") throw new Error("performance.now() should return a number, got: " + typeof t);
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
}

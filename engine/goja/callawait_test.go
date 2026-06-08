package goja_test

import (
	"context"
	"testing"

	"github.com/polagonow/pola/engine/goja"
)

// awaiter is the subset of *goja.Runtime needed to exercise CallAwait with
// dependency injection. NewRuntime returns core.JSRuntime, so the test
// type-asserts to reach the additive methods.
type awaiter interface {
	CallAwait(fn string, args ...any) (any, error)
	SetDependencyInjection(funcs map[string]func(args []any) (any, error)) error
}

const awaitBundle = `
globalThis.syncAction = function () { return JSON.stringify({ ok: true }); };
globalThis.asyncAction = async function () {
  var v = await __DEPENDENCY_INJECTION__.getNum();
  return JSON.stringify({ n: v + 1 });
};
globalThis.rejectAction = async function () { throw new Error("boom"); };
`

func newAwaiter(t *testing.T) awaiter {
	t.Helper()
	eng, err := goja.New(awaitBundle)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	t.Cleanup(rt.Dispose)
	a, ok := rt.(awaiter)
	if !ok {
		t.Fatalf("runtime %T does not implement CallAwait", rt)
	}
	return a
}

func TestCallAwait_Sync(t *testing.T) {
	a := newAwaiter(t)
	res, err := a.CallAwait("syncAction")
	if err != nil {
		t.Fatalf("CallAwait: %v", err)
	}
	if res != `{"ok":true}` {
		t.Errorf("result = %v, want {\"ok\":true}", res)
	}
}

func TestCallAwait_AsyncWithDI(t *testing.T) {
	a := newAwaiter(t)
	if err := a.SetDependencyInjection(map[string]func(args []any) (any, error){
		"getNum": func(_ []any) (any, error) { return 41, nil },
	}); err != nil {
		t.Fatalf("SetDependencyInjection: %v", err)
	}
	res, err := a.CallAwait("asyncAction")
	if err != nil {
		t.Fatalf("CallAwait: %v", err)
	}
	if res != `{"n":42}` {
		t.Errorf("result = %v, want {\"n\":42}", res)
	}
}

func TestCallAwait_Rejection(t *testing.T) {
	a := newAwaiter(t)
	if _, err := a.CallAwait("rejectAction"); err == nil {
		t.Error("expected error from rejected promise")
	}
}

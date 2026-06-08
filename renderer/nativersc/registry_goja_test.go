package nativersc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/polagonow/pola/engine/goja"
	"github.com/polagonow/pola/serveraction"
)

// callAwaiter is the subset of the goja runtime used to drive a returned Promise
// to completion.
type callAwaiter interface {
	CallAwait(fn string, args ...any) (any, error)
}

// buildRegistryBundle assembles a goja-runnable bundle: a fake action module,
// the generated registry installer, a registration call, and the generated
// lookup/invoke helpers — exactly the JS the entry generator emits.
func buildRegistryBundle() string {
	return `
var __sa_0__ = {
  addTodo: async function (title) { return { id: 1, title: title }; },
  boom: async function () { throw new Error("nope"); },
  redirectAction: async function () { return { redirect: "/done" }; },
  throwRedirect: async function () { var e = new Error("r"); e.__pola_redirect__ = "/login"; throw e; },
};
` + serveraction.RegistryJS([]serveraction.Module{{ModuleID: "actions/todo"}})
}

func invoke(t *testing.T, a callAwaiter, id, export string, args ...any) map[string]any {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	res, err := a.CallAwait("__invokeServerAction__", id, export, string(argsJSON))
	if err != nil {
		t.Fatalf("CallAwait(%s:%s): %v", id, export, err)
	}
	s, ok := res.(string)
	if !ok {
		t.Fatalf("envelope is %T, want string", res)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		t.Fatalf("decode envelope %q: %v", s, err)
	}
	return env
}

func TestServerActionRegistry_InvokeInGoja(t *testing.T) {
	eng, err := goja.New(buildRegistryBundle())
	if err != nil {
		t.Fatalf("compile bundle: %v", err)
	}
	rt, err := eng.NewRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Dispose()
	a, ok := rt.(callAwaiter)
	if !ok {
		t.Fatalf("runtime %T lacks CallAwait", rt)
	}

	t.Run("exact key", func(t *testing.T) {
		env := invoke(t, a, "actions/todo", "addTodo", "buy milk")
		if env["success"] != true {
			t.Fatalf("envelope = %v", env)
		}
		result := env["result"].(map[string]any)
		if result["title"] != "buy milk" || result["id"].(float64) != 1 {
			t.Errorf("result = %v", result)
		}
	})

	t.Run("ambiguity-checked suffix match", func(t *testing.T) {
		// Wrong module id, but the export name uniquely identifies the action.
		env := invoke(t, a, "nonexistent/id", "addTodo", "x")
		if env["success"] != true {
			t.Errorf("suffix lookup failed: %v", env)
		}
	})

	t.Run("rejection", func(t *testing.T) {
		env := invoke(t, a, "actions/todo", "boom")
		if env["success"] != false {
			t.Errorf("expected failure, got %v", env)
		}
	})

	t.Run("redirect result", func(t *testing.T) {
		env := invoke(t, a, "actions/todo", "redirectAction")
		if env["success"] != true || env["redirect"] != "/done" {
			t.Errorf("expected redirect, got %v", env)
		}
	})

	t.Run("thrown redirect sentinel", func(t *testing.T) {
		env := invoke(t, a, "actions/todo", "throwRedirect")
		if env["success"] != true || env["redirect"] != "/login" {
			t.Errorf("expected thrown redirect, got %v", env)
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := invoke(t, a, "actions/todo", "missingExport")
		if env["success"] != false {
			t.Errorf("expected not-found failure, got %v", env)
		}
	})
}

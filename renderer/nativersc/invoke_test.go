package nativersc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	gojaengine "github.com/polagonow/pola/engine/goja"
	"github.com/polagonow/pola/serveraction"
)

// actionPoolRenderer builds a nativersc renderer whose VM pool runs a bundle
// containing the server-action registry, so InvokeAction executes for real.
func actionPoolRenderer(t *testing.T) *Renderer {
	t.Helper()
	bundle := `
var __sa_0__ = {
  addTodo: async function (text) { return { ok: true, text: text }; },
  boom: async function () { throw new Error("kaboom"); },
};
` + serveraction.RegistryJS([]serveraction.Module{{ModuleID: "actions/todos"}})

	eng := gojaengine.NewEngine()
	pool, err := eng.NewSSRPool([]byte(bundle))
	if err != nil {
		t.Fatalf("NewSSRPool: %v", err)
	}
	r := New()
	r.pool = pool
	return r
}

func TestInvokeAction_EndToEnd(t *testing.T) {
	r := actionPoolRenderer(t)

	out, err := r.InvokeAction(context.Background(), core.InvokeInput{
		ID:         "actions/todos",
		ExportName: "addTodo",
		Args:       []any{"hello"},
	})
	if err != nil {
		t.Fatalf("InvokeAction: %v", err)
	}
	if !out.Success {
		t.Fatalf("expected success, got %+v", out)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Result, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result["ok"] != true || result["text"] != "hello" {
		t.Errorf("result = %v", result)
	}

	// A throwing action surfaces as an unsuccessful envelope, not a Go error.
	out, err = r.InvokeAction(context.Background(), core.InvokeInput{
		ID: "actions/todos", ExportName: "boom",
	})
	if err != nil {
		t.Fatalf("InvokeAction(boom): %v", err)
	}
	if out.Success || !strings.Contains(out.Error, "kaboom") {
		t.Errorf("expected failure envelope, got %+v", out)
	}
}

// TestServerAction_HTTPToVM exercises the full path: HTTP request → handler
// (CSRF + validation) → renderer InvokeAction → goja → JSON envelope.
func TestServerAction_HTTPToVM(t *testing.T) {
	r := actionPoolRenderer(t)
	h := &serveraction.Handler{
		Invoker:  r,
		ValidIDs: map[string]struct{}{"actions/todos": {}},
	}

	body := `{"id":"actions/todos","export_name":"addTodo","args":["from http"]}`
	req := httptest.NewRequest("POST", "http://example.com/_pola/action", strings.NewReader(body))
	req.Host = "example.com"
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Action(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var env struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if !env.Success || !strings.Contains(string(env.Result), "from http") {
		t.Errorf("envelope = %s", w.Body.String())
	}
}

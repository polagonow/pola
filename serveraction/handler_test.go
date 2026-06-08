package serveraction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
)

type fakeInvoker struct {
	fn func(in core.InvokeInput) (core.InvokeOutput, error)
}

func (f fakeInvoker) InvokeAction(_ context.Context, in core.InvokeInput) (core.InvokeOutput, error) {
	return f.fn(in)
}

func newHandler(inv core.ServerActionInvoker) *Handler {
	return &Handler{
		Invoker:  inv,
		ValidIDs: map[string]struct{}{"actions/todo": {}},
	}
}

func postAction(body string) *http.Request {
	r := httptest.NewRequest("POST", "http://example.com/_pola/action", strings.NewReader(body))
	r.Host = "example.com"
	r.Header.Set("Origin", "http://example.com")
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestAction_Success(t *testing.T) {
	inv := fakeInvoker{fn: func(in core.InvokeInput) (core.InvokeOutput, error) {
		if in.ID != "actions/todo" || in.ExportName != "addTodo" {
			t.Errorf("unexpected invoke: %+v", in)
		}
		return core.InvokeOutput{Success: true, Result: json.RawMessage(`{"id":1}`)}, nil
	}}
	h := newHandler(inv)
	w := httptest.NewRecorder()
	h.Action(w, postAction(`{"id":"actions/todo","export_name":"addTodo","args":["buy milk"]}`))

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !env.Success || string(env.Result) != `{"id":1}` {
		t.Errorf("envelope = %+v", env)
	}
}

func TestAction_Rejections(t *testing.T) {
	inv := fakeInvoker{fn: func(core.InvokeInput) (core.InvokeOutput, error) {
		return core.InvokeOutput{Success: true}, nil
	}}
	h := newHandler(inv)

	cases := []struct {
		name string
		req  *http.Request
		code int
	}{
		{"reserved name", postAction(`{"id":"actions/todo","export_name":"then","args":[]}`), 400},
		{"unknown id", postAction(`{"id":"actions/unknown","export_name":"addTodo","args":[]}`), 404},
		{"bad json", postAction(`not json`), 400},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.Action(w, c.req)
			if w.Code != c.code {
				t.Errorf("status = %d, want %d", w.Code, c.code)
			}
		})
	}

	t.Run("bad origin", func(t *testing.T) {
		r := httptest.NewRequest("POST", "http://example.com/_pola/action", strings.NewReader(`{}`))
		r.Host = "example.com"
		r.Header.Set("Origin", "http://evil.com")
		w := httptest.NewRecorder()
		h.Action(w, r)
		if w.Code != 403 {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})

	t.Run("get not allowed", func(t *testing.T) {
		r := httptest.NewRequest("GET", "http://example.com/_pola/action", nil)
		r.Host = "example.com"
		w := httptest.NewRecorder()
		h.Action(w, r)
		if w.Code != 405 {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestForm_Redirect(t *testing.T) {
	inv := fakeInvoker{fn: func(in core.InvokeInput) (core.InvokeOutput, error) {
		// form action receives (null, formData)
		if len(in.Args) != 2 || in.Args[0] != nil {
			t.Errorf("form args = %+v", in.Args)
		}
		data, _ := in.Args[1].(map[string]any)
		if data["text"] != "hello" {
			t.Errorf("form data = %+v", data)
		}
		return core.InvokeOutput{Success: true}, nil
	}}
	h := newHandler(inv)

	r := httptest.NewRequest("POST", "http://example.com/_pola/form-action",
		strings.NewReader("__action_id=actions/todo&__export_name=addTodo&text=hello"))
	r.Host = "example.com"
	r.Header.Set("Origin", "http://example.com")
	r.Header.Set("Referer", "http://example.com/todos")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Form(w, r)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/todos" {
		t.Errorf("Location = %q, want /todos", loc)
	}
}

package todos

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"

	"todo-api/repositories"
)

// mockTodoService is a hand-rolled mock satisfying
// services.TodoServiceInterface. Each method is backed by a
// *Fn field; unset fields panic on call so missing expectations fail loudly.
type mockTodoService struct {
	createFn func(ctx context.Context, e *repositories.Todo) error
	getFn    func(ctx context.Context, id uint) (*repositories.Todo, error)
	listFn   func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error)
	updateFn func(ctx context.Context, e *repositories.Todo) error
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockTodoService) Create(ctx context.Context, e *repositories.Todo) error {
	if m.createFn == nil {
		panic("createFn not set")
	}
	return m.createFn(ctx, e)
}
func (m *mockTodoService) Get(ctx context.Context, id uint) (*repositories.Todo, error) {
	if m.getFn == nil {
		panic("getFn not set")
	}
	return m.getFn(ctx, id)
}
func (m *mockTodoService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error) {
	if m.listFn == nil {
		panic("listFn not set")
	}
	return m.listFn(ctx, params)
}
func (m *mockTodoService) Update(ctx context.Context, e *repositories.Todo) error {
	if m.updateFn == nil {
		panic("updateFn not set")
	}
	return m.updateFn(ctx, e)
}
func (m *mockTodoService) Delete(ctx context.Context, id uint) error {
	if m.deleteFn == nil {
		panic("deleteFn not set")
	}
	return m.deleteFn(ctx, id)
}

func TestRoute_GET_List(t *testing.T) {
	svc := &mockTodoService{
		listFn: func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error) {
			return &repositories.ListResult[*repositories.Todo]{Page: params.Page, PerPage: params.PerPage}, nil
		},
	}
	r := NewRoute(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/todos", nil)
	r.GET(w, req)
	if w.Code != 200 {
		t.Fatalf("status: got %d want 200; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_POST_BadJSON(t *testing.T) {
	svc := &mockTodoService{}
	r := NewRoute(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/todos", bytes.NewBufferString("not-json"))
	r.POST(w, req)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for bad JSON; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PUT_MissingID(t *testing.T) {
	svc := &mockTodoService{}
	r := NewRoute(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/todos", bytes.NewBufferString(`{}`))
	r.PUT(w, req)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PATCH_MissingID(t *testing.T) {
	svc := &mockTodoService{}
	r := NewRoute(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/todos", bytes.NewBufferString(`{}`))
	r.PATCH(w, req)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_DELETE_MissingID(t *testing.T) {
	svc := &mockTodoService{}
	r := NewRoute(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/todos", nil)
	r.DELETE(w, req)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

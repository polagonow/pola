package todos

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/webframework/std"

	"todo-api/db/models"
	"todo-api/repositories"
	"todo-api/services"
)

// newTestRoute wires the route to the given mock service through a fresh DI
// registry, exercising the real registry-style constructor.
func newTestRoute(svc services.TodoServiceInterface) *Route {
	r := core.NewRegistry()
	core.ProvideValue[services.TodoServiceInterface](r, svc)
	return NewRoute(r)
}

// serve registers h on a std framework and serves a single request, returning
// the recorded response. Handlers take a core.Context, so they cannot be called
// with (w, req) directly.
func serve(method string, body io.Reader, h core.HandlerFunc) *httptest.ResponseRecorder {
	fw := std.New()
	fw.Handle(method, "/todos", h)
	w := httptest.NewRecorder()
	fw.Handler().ServeHTTP(w, httptest.NewRequest(method, "/todos", body))
	return w
}

// mockTodoService is a hand-rolled mock satisfying
// services.TodoServiceInterface. Each method is backed by a
// *Fn field; unset fields panic on call so missing expectations fail loudly.
type mockTodoService struct {
	createFn func(ctx context.Context, e *models.Todo) error
	getFn    func(ctx context.Context, id uint) (*models.Todo, error)
	listFn   func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*models.Todo], error)
	updateFn func(ctx context.Context, e *models.Todo) error
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockTodoService) Create(ctx context.Context, e *models.Todo) error {
	if m.createFn == nil {
		panic("createFn not set")
	}
	return m.createFn(ctx, e)
}
func (m *mockTodoService) Get(ctx context.Context, id uint) (*models.Todo, error) {
	if m.getFn == nil {
		panic("getFn not set")
	}
	return m.getFn(ctx, id)
}
func (m *mockTodoService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*models.Todo], error) {
	if m.listFn == nil {
		panic("listFn not set")
	}
	return m.listFn(ctx, params)
}
func (m *mockTodoService) Update(ctx context.Context, e *models.Todo) error {
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
		listFn: func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*models.Todo], error) {
			return &repositories.ListResult[*models.Todo]{Page: params.Page, PerPage: params.PerPage}, nil
		},
	}
	w := serve("GET", nil, newTestRoute(svc).GET)
	if w.Code != 200 {
		t.Fatalf("status: got %d want 200; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_POST_BadJSON(t *testing.T) {
	w := serve("POST", strings.NewReader("not-json"), newTestRoute(&mockTodoService{}).POST)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for bad JSON; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PUT_MissingID(t *testing.T) {
	w := serve("PUT", strings.NewReader(`{}`), newTestRoute(&mockTodoService{}).PUT)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PATCH_MissingID(t *testing.T) {
	w := serve("PATCH", strings.NewReader(`{}`), newTestRoute(&mockTodoService{}).PATCH)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_DELETE_MissingID(t *testing.T) {
	w := serve("DELETE", nil, newTestRoute(&mockTodoService{}).DELETE)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

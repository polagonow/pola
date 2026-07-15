package tasks

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm-demo/db/models"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	"github.com/polagonow/pola/webframework/std"
)

// mockTaskService is a hand-rolled mock satisfying
// services.TaskServiceInterface. Each method is backed by a
// *Fn field; unset fields panic on call so missing expectations fail loudly.
type mockTaskService struct {
	createFn func(ctx context.Context, e *models.Task) error
	getFn    func(ctx context.Context, id uint) (*models.Task, error)
	listFn   func(ctx context.Context, params repository.ListParams) (*repository.ListResult[*models.Task], error)
	updateFn func(ctx context.Context, e *models.Task) error
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockTaskService) Create(ctx context.Context, e *models.Task) error {
	if m.createFn == nil {
		panic("createFn not set")
	}
	return m.createFn(ctx, e)
}
func (m *mockTaskService) Get(ctx context.Context, id uint) (*models.Task, error) {
	if m.getFn == nil {
		panic("getFn not set")
	}
	return m.getFn(ctx, id)
}
func (m *mockTaskService) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*models.Task], error) {
	if m.listFn == nil {
		panic("listFn not set")
	}
	return m.listFn(ctx, params)
}
func (m *mockTaskService) Update(ctx context.Context, e *models.Task) error {
	if m.updateFn == nil {
		panic("updateFn not set")
	}
	return m.updateFn(ctx, e)
}
func (m *mockTaskService) Delete(ctx context.Context, id uint) error {
	if m.deleteFn == nil {
		panic("deleteFn not set")
	}
	return m.deleteFn(ctx, id)
}

// serve registers h on a std framework and serves a single request.
func serve(method string, body io.Reader, h core.HandlerFunc) *httptest.ResponseRecorder {
	fw := std.New()
	fw.Handle(method, "/tasks", h)
	w := httptest.NewRecorder()
	fw.Handler().ServeHTTP(w, httptest.NewRequest(method, "/tasks", body))
	return w
}

func TestRoute_GET_List(t *testing.T) {
	svc := &mockTaskService{
		listFn: func(ctx context.Context, params repository.ListParams) (*repository.ListResult[*models.Task], error) {
			return &repository.ListResult[*models.Task]{Page: params.Page, PerPage: params.PerPage}, nil
		},
	}
	w := serve("GET", nil, NewRoute(svc).GET)
	if w.Code != 200 {
		t.Fatalf("status: got %d want 200; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_POST_BadJSON(t *testing.T) {
	w := serve("POST", strings.NewReader("not-json"), NewRoute(&mockTaskService{}).POST)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for bad JSON; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_PUT_MissingID(t *testing.T) {
	w := serve("PUT", strings.NewReader(`{}`), NewRoute(&mockTaskService{}).PUT)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

func TestRoute_DELETE_MissingID(t *testing.T) {
	w := serve("DELETE", nil, NewRoute(&mockTaskService{}).DELETE)
	if w.Code != 400 {
		t.Fatalf("status: got %d want 400 for missing id; body=%q", w.Code, w.Body.String())
	}
}

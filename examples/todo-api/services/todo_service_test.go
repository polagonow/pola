package services

import (
	"context"
	"errors"
	"testing"

	"github.com/polagonow/pola/core"
	"todo-api/repositories"
)

// newTestTodoService wires the service to the given mock repository through a
// fresh DI registry, exercising the real registry-style constructor.
func newTestTodoService(repo repositories.TodoRepository) *TodoService {
	r := core.NewRegistry()
	core.ProvideValue[repositories.TodoRepository](r, repo)
	return NewTodoService(r)
}

// mockTodoRepository is a hand-rolled mock implementing
// repositories.TodoRepository. Each method is backed by a *Fn field that
// tests set to control behavior. Unset fields panic on call so missing
// expectations fail loudly.
type mockTodoRepository struct {
	createFn func(ctx context.Context, e *repositories.Todo) error
	getFn    func(ctx context.Context, id uint) (*repositories.Todo, error)
	listFn   func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error)
	updateFn func(ctx context.Context, e *repositories.Todo) error
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockTodoRepository) Create(ctx context.Context, e *repositories.Todo) error {
	if m.createFn == nil {
		panic("createFn not set")
	}
	return m.createFn(ctx, e)
}
func (m *mockTodoRepository) Get(ctx context.Context, id uint) (*repositories.Todo, error) {
	if m.getFn == nil {
		panic("getFn not set")
	}
	return m.getFn(ctx, id)
}
func (m *mockTodoRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error) {
	if m.listFn == nil {
		panic("listFn not set")
	}
	return m.listFn(ctx, params)
}
func (m *mockTodoRepository) Update(ctx context.Context, e *repositories.Todo) error {
	if m.updateFn == nil {
		panic("updateFn not set")
	}
	return m.updateFn(ctx, e)
}
func (m *mockTodoRepository) Delete(ctx context.Context, id uint) error {
	if m.deleteFn == nil {
		panic("deleteFn not set")
	}
	return m.deleteFn(ctx, id)
}

func TestTodoService_Create_DelegatesToRepo(t *testing.T) {
	called := false
	repo := &mockTodoRepository{
		createFn: func(ctx context.Context, e *repositories.Todo) error {
			called = true
			return nil
		},
	}
	svc := newTestTodoService(repo)
	if err := svc.Create(context.Background(), &repositories.Todo{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Create to be called")
	}
}

func TestTodoService_Create_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	repo := &mockTodoRepository{
		createFn: func(ctx context.Context, e *repositories.Todo) error { return want },
	}
	svc := newTestTodoService(repo)
	if err := svc.Create(context.Background(), &repositories.Todo{}); !errors.Is(err, want) {
		t.Fatalf("got err %v, want %v", err, want)
	}
}

func TestTodoService_Get_DelegatesToRepo(t *testing.T) {
	want := &repositories.Todo{}
	repo := &mockTodoRepository{
		getFn: func(ctx context.Context, id uint) (*repositories.Todo, error) {
			return want, nil
		},
	}
	svc := newTestTodoService(repo)
	got, err := svc.Get(context.Background(), 0)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTodoService_List_DelegatesToRepo(t *testing.T) {
	want := &repositories.ListResult[*repositories.Todo]{Page: 1, PerPage: 10}
	repo := &mockTodoRepository{
		listFn: func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error) {
			return want, nil
		},
	}
	svc := newTestTodoService(repo)
	got, err := svc.List(context.Background(), repositories.ListParams{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTodoService_Update_DelegatesToRepo(t *testing.T) {
	called := false
	repo := &mockTodoRepository{
		updateFn: func(ctx context.Context, e *repositories.Todo) error {
			called = true
			return nil
		},
	}
	svc := newTestTodoService(repo)
	if err := svc.Update(context.Background(), &repositories.Todo{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Update to be called")
	}
}

func TestTodoService_Delete_DelegatesToRepo(t *testing.T) {
	called := false
	repo := &mockTodoRepository{
		deleteFn: func(ctx context.Context, id uint) error {
			called = true
			return nil
		},
	}
	svc := newTestTodoService(repo)
	if err := svc.Delete(context.Background(), 0); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Delete to be called")
	}
}

// Verify *TodoService satisfies TodoServiceInterface at compile time.
var _ TodoServiceInterface = (*TodoService)(nil)

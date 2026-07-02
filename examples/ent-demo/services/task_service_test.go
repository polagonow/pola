package services

import (
	"context"
	"errors"
	"testing"

	"ent-demo/repositories"
)

// mockTaskRepository is a hand-rolled mock implementing
// repositories.TaskRepository. Each method is backed by a *Fn field that
// tests set to control behavior. Unset fields panic on call so missing
// expectations fail loudly.
type mockTaskRepository struct {
	createFn func(ctx context.Context, e *repositories.Task) error
	getFn    func(ctx context.Context, id uint) (*repositories.Task, error)
	listFn   func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Task], error)
	updateFn func(ctx context.Context, e *repositories.Task) error
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockTaskRepository) Create(ctx context.Context, e *repositories.Task) error {
	if m.createFn == nil {
		panic("createFn not set")
	}
	return m.createFn(ctx, e)
}
func (m *mockTaskRepository) Get(ctx context.Context, id uint) (*repositories.Task, error) {
	if m.getFn == nil {
		panic("getFn not set")
	}
	return m.getFn(ctx, id)
}
func (m *mockTaskRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Task], error) {
	if m.listFn == nil {
		panic("listFn not set")
	}
	return m.listFn(ctx, params)
}
func (m *mockTaskRepository) Update(ctx context.Context, e *repositories.Task) error {
	if m.updateFn == nil {
		panic("updateFn not set")
	}
	return m.updateFn(ctx, e)
}
func (m *mockTaskRepository) Delete(ctx context.Context, id uint) error {
	if m.deleteFn == nil {
		panic("deleteFn not set")
	}
	return m.deleteFn(ctx, id)
}

func TestTaskService_Create_DelegatesToRepo(t *testing.T) {
	called := false
	repo := &mockTaskRepository{
		createFn: func(ctx context.Context, e *repositories.Task) error {
			called = true
			return nil
		},
	}
	svc := NewTaskService(repo)
	if err := svc.Create(context.Background(), &repositories.Task{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Create to be called")
	}
}

func TestTaskService_Create_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	repo := &mockTaskRepository{
		createFn: func(ctx context.Context, e *repositories.Task) error { return want },
	}
	svc := NewTaskService(repo)
	if err := svc.Create(context.Background(), &repositories.Task{}); !errors.Is(err, want) {
		t.Fatalf("got err %v, want %v", err, want)
	}
}

func TestTaskService_Get_DelegatesToRepo(t *testing.T) {
	want := &repositories.Task{}
	repo := &mockTaskRepository{
		getFn: func(ctx context.Context, id uint) (*repositories.Task, error) {
			return want, nil
		},
	}
	svc := NewTaskService(repo)
	got, err := svc.Get(context.Background(), 0)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTaskService_List_DelegatesToRepo(t *testing.T) {
	want := &repositories.ListResult[*repositories.Task]{Page: 1, PerPage: 10}
	repo := &mockTaskRepository{
		listFn: func(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Task], error) {
			return want, nil
		},
	}
	svc := NewTaskService(repo)
	got, err := svc.List(context.Background(), repositories.ListParams{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTaskService_Update_DelegatesToRepo(t *testing.T) {
	called := false
	repo := &mockTaskRepository{
		updateFn: func(ctx context.Context, e *repositories.Task) error {
			called = true
			return nil
		},
	}
	svc := NewTaskService(repo)
	if err := svc.Update(context.Background(), &repositories.Task{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Update to be called")
	}
}

func TestTaskService_Delete_DelegatesToRepo(t *testing.T) {
	called := false
	repo := &mockTaskRepository{
		deleteFn: func(ctx context.Context, id uint) error {
			called = true
			return nil
		},
	}
	svc := NewTaskService(repo)
	if err := svc.Delete(context.Background(), 0); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !called {
		t.Fatal("expected repo.Delete to be called")
	}
}

// Verify *TaskService satisfies TaskServiceInterface at compile time.
var _ TaskServiceInterface = (*TaskService)(nil)

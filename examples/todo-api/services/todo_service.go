package services

import (
	"context"

	"todo-api/repositories"
)

// TodoServiceInterface is the contract a Todo service must satisfy.
// Routes and other call sites depend on this interface so tests can substitute
// mocks. The concrete TodoService type below satisfies it.
type TodoServiceInterface interface {
	Create(ctx context.Context, entity *repositories.Todo) error
	Get(ctx context.Context, id uint) (*repositories.Todo, error)
	List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error)
	Update(ctx context.Context, entity *repositories.Todo) error
	Delete(ctx context.Context, id uint) error
}

// TodoService handles business logic for todo operations.
type TodoService struct {
	repo repositories.TodoRepository
}

// NewTodoService creates a new TodoService.
func NewTodoService(repo repositories.TodoRepository) *TodoService {
	return &TodoService{repo: repo}
}

// Create creates a new todo.
func (s *TodoService) Create(ctx context.Context, entity *repositories.Todo) error {
	// TODO: add business logic / validation
	return s.repo.Create(ctx, entity)
}

// Get returns a todo by its ID.
func (s *TodoService) Get(ctx context.Context, id uint) (*repositories.Todo, error) {
	return s.repo.Get(ctx, id)
}

// List returns a paginated list of todos.
func (s *TodoService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error) {
	return s.repo.List(ctx, params)
}

// Update updates an existing todo.
func (s *TodoService) Update(ctx context.Context, entity *repositories.Todo) error {
	// TODO: add business logic / validation
	return s.repo.Update(ctx, entity)
}

// Delete removes a todo by its ID.
func (s *TodoService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

package services

import (
	"context"

	"gorm-demo/repositories"
)

// TaskServiceInterface is the contract a Task service must satisfy.
// Routes and other call sites depend on this interface so tests can substitute
// mocks. The concrete TaskService type below satisfies it.
type TaskServiceInterface interface {
	Create(ctx context.Context, entity *repositories.Task) error
	Get(ctx context.Context, id uint) (*repositories.Task, error)
	List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Task], error)
	Update(ctx context.Context, entity *repositories.Task) error
	Delete(ctx context.Context, id uint) error
}

// TaskService handles business logic for task operations.
type TaskService struct {
	repo repositories.TaskRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService(repo repositories.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// Create creates a new task.
func (s *TaskService) Create(ctx context.Context, entity *repositories.Task) error {
	// TODO: add business logic / validation
	return s.repo.Create(ctx, entity)
}

// Get returns a task by its ID.
func (s *TaskService) Get(ctx context.Context, id uint) (*repositories.Task, error) {
	return s.repo.Get(ctx, id)
}

// List returns a paginated list of tasks.
func (s *TaskService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Task], error) {
	return s.repo.List(ctx, params)
}

// Update updates an existing task.
func (s *TaskService) Update(ctx context.Context, entity *repositories.Task) error {
	// TODO: add business logic / validation
	return s.repo.Update(ctx, entity)
}

// Delete removes a task by its ID.
func (s *TaskService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

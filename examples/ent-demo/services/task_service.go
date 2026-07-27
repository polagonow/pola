package services

import (
	"ent-demo/db/models"
	"ent-demo/repositories"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"
)

// TaskServiceInterface is the contract for task business logic. It embeds
// the framework's standard CRUD service; add custom business methods here.
// Routes and other call sites depend on this interface.
type TaskServiceInterface interface {
	service.Service[models.Task, uint]
}

// TaskService handles business logic for task operations. The embedded
// generic service delegates CRUD to the repository; override a method (e.g.
// Create) on this struct to add validation or business rules, using s.repo.
type TaskService struct {
	service.Service[models.Task, uint]
	repo repositories.TaskRepository
}

// NewTaskService creates a new TaskService, resolving its dependencies from
// the DI registry.
func NewTaskService(r *core.Registry) *TaskService {
	repo := core.MustInvoke[repositories.TaskRepository](r)
	return &TaskService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ TaskServiceInterface = (*TaskService)(nil)

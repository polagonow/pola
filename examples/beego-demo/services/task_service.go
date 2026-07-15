package services

import (
	"beego-demo/db/models"
	"beego-demo/repositories"

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

// NewTaskService creates a new TaskService.
func NewTaskService(repo repositories.TaskRepository) *TaskService {
	return &TaskService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ TaskServiceInterface = (*TaskService)(nil)

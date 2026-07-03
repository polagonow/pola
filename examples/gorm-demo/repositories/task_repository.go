package repositories

import (
	"github.com/polagonow/pola/repository"
)

// Task represents the task entity for the repository layer.
type Task struct {
	ID    uint   `json:"id"`
	Title string `json:"title" validate:"required"`
	Done  bool   `json:"done" validate:"-"`
}

// TaskRepository defines the contract for task persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type TaskRepository interface {
	repository.Repository[Task, uint]
}

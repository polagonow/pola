package repositories

import (
	"github.com/polagonow/pola/repository"
)

// Task represents the task entity for the repository layer.
type Task struct {
	ID    uint   `orm:"column(id);auto;pk" json:"id"`
	Title string `orm:"column(title)" json:"title" validate:"required"`
	Done  bool   `orm:"column(done)" json:"done" validate:"-"`
}

// TableName maps Task to the tasks table, matching migrations.
func (Task) TableName() string { return "tasks" }

// TaskRepository defines the contract for task persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type TaskRepository interface {
	repository.Repository[Task, uint]
}

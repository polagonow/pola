package repositories

import (
	"github.com/polagonow/pola/repository"

	"gorm-demo/db/models"
)

// TaskRepository defines the contract for task persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type TaskRepository interface {
	repository.Repository[models.Task, uint]
}

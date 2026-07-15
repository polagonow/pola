package repositories

import (
	"github.com/polagonow/pola/repository"

	"beego-demo/db/models"
)

// TaskRepository defines the contract for task persistence operations. The
// entity type is the canonical model in db/models — the single source of truth.
// It embeds the framework's standard CRUD contract; add custom query methods here.
type TaskRepository interface {
	repository.Repository[models.Task, uint]
}

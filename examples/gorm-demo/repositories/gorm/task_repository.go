package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"gorm-demo/repositories"
)

// taskRepository is the GORM-backed TaskRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type taskRepository struct {
	repository.Repository[repositories.Task, uint]
	db *gorm.DB
}

// NewTaskRepository creates a new GORM-backed TaskRepository.
func NewTaskRepository(db *gorm.DB) repositories.TaskRepository {
	return &taskRepository{
		Repository: gormrepo.New[repositories.Task, uint](db),
		db:         db,
	}
}

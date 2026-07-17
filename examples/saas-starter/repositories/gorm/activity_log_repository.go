package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// activityLogRepository is the GORM-backed ActivityLogRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete
// (including framework-owned timestamps and soft-delete); add custom queries
// as methods on this struct using r.db.
type activityLogRepository struct {
	repository.Repository[models.ActivityLog, uint]
	db *gorm.DB
}

// NewActivityLogRepository creates a new GORM-backed ActivityLogRepository,
// resolving its dependencies from the DI registry.
func NewActivityLogRepository(r *core.Registry) repositories.ActivityLogRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &activityLogRepository{
		Repository: gormrepo.New[models.ActivityLog, uint](db),
		db:         db,
	}
}

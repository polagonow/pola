package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// teamRepository is the GORM-backed TeamRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete
// (including framework-owned timestamps and soft-delete); add custom queries
// as methods on this struct using r.db.
type teamRepository struct {
	repository.Repository[models.Team, uint]
	db *gorm.DB
}

// NewTeamRepository creates a new GORM-backed TeamRepository,
// resolving its dependencies from the DI registry.
func NewTeamRepository(r *core.Registry) repositories.TeamRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &teamRepository{
		Repository: gormrepo.New[models.Team, uint](db),
		db:         db,
	}
}

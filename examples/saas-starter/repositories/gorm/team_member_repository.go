package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// teamMemberRepository is the GORM-backed TeamMemberRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete
// (including framework-owned timestamps and soft-delete); add custom queries
// as methods on this struct using r.db.
type teamMemberRepository struct {
	repository.Repository[models.TeamMember, uint]
	db *gorm.DB
}

// NewTeamMemberRepository creates a new GORM-backed TeamMemberRepository,
// resolving its dependencies from the DI registry.
func NewTeamMemberRepository(r *core.Registry) repositories.TeamMemberRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &teamMemberRepository{
		Repository: gormrepo.New[models.TeamMember, uint](db),
		db:         db,
	}
}

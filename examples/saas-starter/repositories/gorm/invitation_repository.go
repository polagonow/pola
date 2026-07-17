package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// invitationRepository is the GORM-backed InvitationRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete
// (including framework-owned timestamps and soft-delete); add custom queries
// as methods on this struct using r.db.
type invitationRepository struct {
	repository.Repository[models.Invitation, uint]
	db *gorm.DB
}

// NewInvitationRepository creates a new GORM-backed InvitationRepository,
// resolving its dependencies from the DI registry.
func NewInvitationRepository(r *core.Registry) repositories.InvitationRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &invitationRepository{
		Repository: gormrepo.New[models.Invitation, uint](db),
		db:         db,
	}
}

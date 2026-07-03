package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"validation/repositories"
)

// contactRepository is the GORM-backed ContactRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type contactRepository struct {
	repository.Repository[repositories.Contact, uint]
	db *gorm.DB
}

// NewContactRepository creates a new GORM-backed ContactRepository, resolving
// its dependencies from the DI registry.
func NewContactRepository(r *core.Registry) repositories.ContactRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &contactRepository{
		Repository: gormrepo.New[repositories.Contact, uint](db),
		db:         db,
	}
}

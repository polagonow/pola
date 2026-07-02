package gorm

import (
	"gorm.io/gorm"

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

// NewContactRepository creates a new GORM-backed ContactRepository.
func NewContactRepository(db *gorm.DB) repositories.ContactRepository {
	return &contactRepository{
		Repository: gormrepo.New[repositories.Contact, uint](db),
		db:         db,
	}
}

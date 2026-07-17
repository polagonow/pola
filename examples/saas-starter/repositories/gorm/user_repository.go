package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// userRepository is the GORM-backed UserRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete
// (including framework-owned timestamps and soft-delete); add custom queries
// as methods on this struct using r.db.
type userRepository struct {
	repository.Repository[models.User, uint]
	db *gorm.DB
}

// NewUserRepository creates a new GORM-backed UserRepository,
// resolving its dependencies from the DI registry.
func NewUserRepository(r *core.Registry) repositories.UserRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &userRepository{
		Repository: gormrepo.New[models.User, uint](db),
		db:         db,
	}
}

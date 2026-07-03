package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"validation/repositories"
)

// serverRepository is the GORM-backed ServerRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type serverRepository struct {
	repository.Repository[repositories.Server, uint]
	db *gorm.DB
}

// NewServerRepository creates a new GORM-backed ServerRepository, resolving
// its dependencies from the DI registry.
func NewServerRepository(r *core.Registry) repositories.ServerRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &serverRepository{
		Repository: gormrepo.New[repositories.Server, uint](db),
		db:         db,
	}
}

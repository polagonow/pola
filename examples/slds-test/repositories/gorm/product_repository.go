package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"slds-test/repositories"
)

// productRepository is the GORM-backed ProductRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type productRepository struct {
	repository.Repository[repositories.Product, uint]
	db *gorm.DB
}

// NewProductRepository creates a new GORM-backed ProductRepository, resolving
// its dependencies from the DI registry.
func NewProductRepository(r *core.Registry) repositories.ProductRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &productRepository{
		Repository: gormrepo.New[repositories.Product, uint](db),
		db:         db,
	}
}

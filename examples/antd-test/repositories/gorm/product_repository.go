package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"antd-test/repositories"
)

// productRepository is the GORM-backed ProductRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type productRepository struct {
	repository.Repository[repositories.Product, uint]
	db *gorm.DB
}

// NewProductRepository creates a new GORM-backed ProductRepository.
func NewProductRepository(db *gorm.DB) repositories.ProductRepository {
	return &productRepository{
		Repository: gormrepo.New[repositories.Product, uint](db),
		db:         db,
	}
}

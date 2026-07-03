package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"blog-e2e-react/repositories"
)

// sampleEntityRepository is the GORM-backed SampleEntityRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type sampleEntityRepository struct {
	repository.Repository[repositories.SampleEntity, uint]
	db *gorm.DB
}

// NewSampleEntityRepository creates a new GORM-backed SampleEntityRepository.
func NewSampleEntityRepository(db *gorm.DB) repositories.SampleEntityRepository {
	return &sampleEntityRepository{
		Repository: gormrepo.New[repositories.SampleEntity, uint](db),
		db:         db,
	}
}

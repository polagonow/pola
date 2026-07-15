package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"blog-e2e-react/db/models"
	"blog-e2e-react/repositories"
)

// sampleEntityRepository is the GORM-backed SampleEntityRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type sampleEntityRepository struct {
	repository.Repository[models.SampleEntity, uint]
	db *gorm.DB
}

// NewSampleEntityRepository creates a new GORM-backed SampleEntityRepository,
// resolving its dependencies from the DI registry.
func NewSampleEntityRepository(r *core.Registry) repositories.SampleEntityRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &sampleEntityRepository{
		Repository: gormrepo.New[models.SampleEntity, uint](db),
		db:         db,
	}
}

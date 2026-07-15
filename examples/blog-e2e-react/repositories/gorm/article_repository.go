package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"blog-e2e-react/db/models"
	"blog-e2e-react/repositories"
)

// articleRepository is the GORM-backed ArticleRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type articleRepository struct {
	repository.Repository[models.Article, uint]
	db *gorm.DB
}

// NewArticleRepository creates a new GORM-backed ArticleRepository, resolving
// its dependencies from the DI registry.
func NewArticleRepository(r *core.Registry) repositories.ArticleRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &articleRepository{
		Repository: gormrepo.New[models.Article, uint](db),
		db:         db,
	}
}

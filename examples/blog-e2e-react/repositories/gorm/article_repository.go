package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"blog-e2e-react/repositories"
)

// articleRepository is the GORM-backed ArticleRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type articleRepository struct {
	repository.Repository[repositories.Article, uint]
	db *gorm.DB
}

// NewArticleRepository creates a new GORM-backed ArticleRepository.
func NewArticleRepository(db *gorm.DB) repositories.ArticleRepository {
	return &articleRepository{
		Repository: gormrepo.New[repositories.Article, uint](db),
		db:         db,
	}
}

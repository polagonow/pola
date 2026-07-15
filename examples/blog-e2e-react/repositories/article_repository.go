package repositories

import (
	"github.com/polagonow/pola/repository"

	"blog-e2e-react/db/models"
)

// ArticleRepository defines the contract for article persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type ArticleRepository interface {
	repository.Repository[models.Article, uint]
}

package repositories

import (
	"github.com/polagonow/pola/repository"
)

// Article represents the article entity for the repository layer.
type Article struct {
	ID    uint   `json:"id"`
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"required"`
}

// ArticleRepository defines the contract for article persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type ArticleRepository interface {
	repository.Repository[Article, uint]
}

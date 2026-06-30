package repositories

import (
	"context"
)

// Article represents the article entity for the repository layer.
type Article struct {
	ID    uint   `json:"id"`
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"required"`
}

// ArticleRepository defines the contract for article persistence operations.
type ArticleRepository interface {
	Create(ctx context.Context, entity *Article) error
	Get(ctx context.Context, id uint) (*Article, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Article], error)
	Update(ctx context.Context, entity *Article) error
	Delete(ctx context.Context, id uint) error
}

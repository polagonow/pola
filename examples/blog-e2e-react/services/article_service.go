package services

import (
	"context"

	"blog-e2e-react/repositories"
)

// ArticleService handles business logic for article operations.
type ArticleService struct {
	repo repositories.ArticleRepository
}

// NewArticleService creates a new ArticleService.
func NewArticleService(repo repositories.ArticleRepository) *ArticleService {
	return &ArticleService{repo: repo}
}

// Create creates a new article.
func (s *ArticleService) Create(ctx context.Context, entity *repositories.Article) error {
	// TODO: add business logic / validation
	return s.repo.Create(ctx, entity)
}

// Get returns a article by its ID.
func (s *ArticleService) Get(ctx context.Context, id uint) (*repositories.Article, error) {
	return s.repo.Get(ctx, id)
}

// List returns a paginated list of articles.
func (s *ArticleService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Article], error) {
	return s.repo.List(ctx, params)
}

// Update updates an existing article.
func (s *ArticleService) Update(ctx context.Context, entity *repositories.Article) error {
	// TODO: add business logic / validation
	return s.repo.Update(ctx, entity)
}

// Delete removes a article by its ID.
func (s *ArticleService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

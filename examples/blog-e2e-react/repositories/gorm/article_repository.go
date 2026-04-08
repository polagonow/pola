package gorm

import (
	"context"
	"fmt"

	"blog-e2e-react/repositories"
	"gorm.io/gorm"
)

type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository creates a new GORM-backed ArticleRepository.
func NewArticleRepository(db *gorm.DB) repositories.ArticleRepository {
	return &articleRepository{db: db}
}

func (r *articleRepository) Create(ctx context.Context, entity *repositories.Article) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *articleRepository) Get(ctx context.Context, id uint) (*repositories.Article, error) {
	var entity repositories.Article
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get article by id: %w", err)
	}
	return &entity, nil
}

func (r *articleRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Article], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.Article{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count articles: %w", err)
	}

	var items []*repositories.Article
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Article]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *articleRepository) Update(ctx context.Context, entity *repositories.Article) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *articleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.Article{}, id).Error
}

package gorm

import (
	"context"
	"fmt"

	"carbon-tf/repositories"
	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

// NewProductRepository creates a new GORM-backed ProductRepository.
func NewProductRepository(db *gorm.DB) repositories.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, entity *repositories.Product) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *productRepository) Get(ctx context.Context, id uint) (*repositories.Product, error) {
	var entity repositories.Product
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return &entity, nil
}

func (r *productRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Product], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.Product{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count products: %w", err)
	}

	var items []*repositories.Product
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Product]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *productRepository) Update(ctx context.Context, entity *repositories.Product) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *productRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.Product{}, id).Error
}

package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"mcp-hello/repositories"
)

type greetingRepository struct {
	db *gorm.DB
}

// NewGreetingRepository creates a new GORM-backed GreetingRepository.
func NewGreetingRepository(db *gorm.DB) repositories.GreetingRepository {
	return &greetingRepository{db: db}
}

func (r *greetingRepository) Create(ctx context.Context, entity *repositories.Greeting) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *greetingRepository) Get(ctx context.Context, id uint) (*repositories.Greeting, error) {
	var entity repositories.Greeting
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get greeting by id: %w", err)
	}
	return &entity, nil
}

func (r *greetingRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Greeting], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.Greeting{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count greetings: %w", err)
	}

	var items []*repositories.Greeting
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list greetings: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Greeting]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *greetingRepository) Update(ctx context.Context, entity *repositories.Greeting) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *greetingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.Greeting{}, id).Error
}

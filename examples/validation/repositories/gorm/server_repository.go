package gorm

import (
	"context"
	"fmt"

	"validation/repositories"
	"gorm.io/gorm"
)

type serverRepository struct {
	db *gorm.DB
}

// NewServerRepository creates a new GORM-backed ServerRepository.
func NewServerRepository(db *gorm.DB) repositories.ServerRepository {
	return &serverRepository{db: db}
}

func (r *serverRepository) Create(ctx context.Context, entity *repositories.Server) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *serverRepository) Get(ctx context.Context, id uint) (*repositories.Server, error) {
	var entity repositories.Server
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get server by id: %w", err)
	}
	return &entity, nil
}

func (r *serverRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Server], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.Server{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count servers: %w", err)
	}

	var items []*repositories.Server
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Server]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *serverRepository) Update(ctx context.Context, entity *repositories.Server) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *serverRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.Server{}, id).Error
}

package gorm

import (
	"context"
	"fmt"

	"blog-e2e-react/repositories"
	"gorm.io/gorm"
)

type sampleEntityRepository struct {
	db *gorm.DB
}

// NewSampleEntityRepository creates a new GORM-backed SampleEntityRepository.
func NewSampleEntityRepository(db *gorm.DB) repositories.SampleEntityRepository {
	return &sampleEntityRepository{db: db}
}

func (r *sampleEntityRepository) Create(ctx context.Context, entity *repositories.SampleEntity) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *sampleEntityRepository) Get(ctx context.Context, id uint) (*repositories.SampleEntity, error) {
	var entity repositories.SampleEntity
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get sample_entity by id: %w", err)
	}
	return &entity, nil
}

func (r *sampleEntityRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.SampleEntity], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.SampleEntity{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count sample_entities: %w", err)
	}

	var items []*repositories.SampleEntity
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list sample_entities: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.SampleEntity]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *sampleEntityRepository) Update(ctx context.Context, entity *repositories.SampleEntity) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *sampleEntityRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.SampleEntity{}, id).Error
}

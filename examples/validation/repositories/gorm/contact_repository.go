package gorm

import (
	"context"
	"fmt"

	"validation/repositories"
	"gorm.io/gorm"
)

type contactRepository struct {
	db *gorm.DB
}

// NewContactRepository creates a new GORM-backed ContactRepository.
func NewContactRepository(db *gorm.DB) repositories.ContactRepository {
	return &contactRepository{db: db}
}

func (r *contactRepository) Create(ctx context.Context, entity *repositories.Contact) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *contactRepository) Get(ctx context.Context, id uint) (*repositories.Contact, error) {
	var entity repositories.Contact
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get contact by id: %w", err)
	}
	return &entity, nil
}

func (r *contactRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Contact], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.Contact{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count contacts: %w", err)
	}

	var items []*repositories.Contact
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Contact]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *contactRepository) Update(ctx context.Context, entity *repositories.Contact) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *contactRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.Contact{}, id).Error
}

package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"todo-api/repositories"
)

type todoRepository struct {
	db *gorm.DB
}

// NewTodoRepository creates a new GORM-backed TodoRepository.
func NewTodoRepository(db *gorm.DB) repositories.TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, entity *repositories.Todo) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *todoRepository) Get(ctx context.Context, id uint) (*repositories.Todo, error) {
	var entity repositories.Todo
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get todo by id: %w", err)
	}
	return &entity, nil
}

func (r *todoRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Todo], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&repositories.Todo{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count todos: %w", err)
	}

	var items []*repositories.Todo
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*repositories.Todo]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *todoRepository) Update(ctx context.Context, entity *repositories.Todo) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *todoRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&repositories.Todo{}, id).Error
}

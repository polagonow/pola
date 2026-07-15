package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"todo-api/db/models"
	"todo-api/repositories"
)

type todoRepository struct {
	db *gorm.DB
}

// NewTodoRepository creates a new GORM-backed TodoRepository, resolving its
// dependencies from the DI registry.
func NewTodoRepository(r *core.Registry) repositories.TodoRepository {
	return &todoRepository{db: core.MustInvoke[*gorm.DB](r)}
}

func (r *todoRepository) Create(ctx context.Context, entity *models.Todo) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *todoRepository) Get(ctx context.Context, id uint) (*models.Todo, error) {
	var entity models.Todo
	if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
		return nil, fmt.Errorf("get todo by id: %w", err)
	}
	return &entity, nil
}

func (r *todoRepository) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*models.Todo], error) {
	params = params.Normalize()

	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Todo{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count todos: %w", err)
	}

	var items []*models.Todo
	if err := r.db.WithContext(ctx).Offset(params.Offset()).Limit(params.PerPage).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}

	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}

	return &repositories.ListResult[*models.Todo]{
		Items:      items,
		Total:      int(total),
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (r *todoRepository) Update(ctx context.Context, entity *models.Todo) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *todoRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Todo{}, id).Error
}

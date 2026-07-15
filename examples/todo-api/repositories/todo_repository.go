package repositories

import (
	"context"

	"todo-api/db/models"
)

// TodoRepository defines the contract for todo persistence operations.
type TodoRepository interface {
	Create(ctx context.Context, entity *models.Todo) error
	Get(ctx context.Context, id uint) (*models.Todo, error)
	List(ctx context.Context, params ListParams) (*ListResult[*models.Todo], error)
	Update(ctx context.Context, entity *models.Todo) error
	Delete(ctx context.Context, id uint) error
}

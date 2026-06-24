package repositories

import (
	"context"
)

// Todo represents the todo entity for the repository layer.
type Todo struct {
	ID        uint   `json:"id"`
	Title     string `json:"title" valid:"required"`
	Completed bool   `json:"completed" valid:"-"`
}

// TodoRepository defines the contract for todo persistence operations.
type TodoRepository interface {
	Create(ctx context.Context, entity *Todo) error
	Get(ctx context.Context, id uint) (*Todo, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Todo], error)
	Update(ctx context.Context, entity *Todo) error
	Delete(ctx context.Context, id uint) error
}

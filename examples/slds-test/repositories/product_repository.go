package repositories

import (
	"context"
)

// Product represents the product entity for the repository layer.
type Product struct {
	ID     uint   `json:"id"`
	Name   string `json:"name" validate:"required"`
	Amount int    `json:"amount" validate:"required"`
}

// ProductRepository defines the contract for product persistence operations.
type ProductRepository interface {
	Create(ctx context.Context, entity *Product) error
	Get(ctx context.Context, id uint) (*Product, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Product], error)
	Update(ctx context.Context, entity *Product) error
	Delete(ctx context.Context, id uint) error
}

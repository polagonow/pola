package repositories

import (
	"context"
)

// Greeting represents the greeting entity for the repository layer.
type Greeting struct {
	ID      uint   `json:"id"`
	Message string `json:"message" valid:"required"`
}

// GreetingRepository defines the contract for greeting persistence operations.
type GreetingRepository interface {
	Create(ctx context.Context, entity *Greeting) error
	Get(ctx context.Context, id uint) (*Greeting, error)
	List(ctx context.Context, params ListParams) (*ListResult[*Greeting], error)
	Update(ctx context.Context, entity *Greeting) error
	Delete(ctx context.Context, id uint) error
}

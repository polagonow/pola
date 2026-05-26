package repositories

import (
	"context"
)

// SampleEntity represents the sample_entity entity for the repository layer.
type SampleEntity struct {
	ID   uint   `json:"id"`
	Name string `json:"name" valid:"required"`
}

// SampleEntityRepository defines the contract for sample_entity persistence operations.
type SampleEntityRepository interface {
	Create(ctx context.Context, entity *SampleEntity) error
	Get(ctx context.Context, id uint) (*SampleEntity, error)
	List(ctx context.Context, params ListParams) (*ListResult[*SampleEntity], error)
	Update(ctx context.Context, entity *SampleEntity) error
	Delete(ctx context.Context, id uint) error
}

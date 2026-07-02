package repositories

import (
	"github.com/polagonow/pola/repository"
)

// SampleEntity represents the sample_entity entity for the repository layer.
type SampleEntity struct {
	ID   uint   `json:"id"`
	Name string `json:"name" validate:"required"`
}

// SampleEntityRepository defines the contract for sample_entity persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type SampleEntityRepository interface {
	repository.Repository[SampleEntity, uint]
}

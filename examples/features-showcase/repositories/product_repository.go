package repositories

import (
	"github.com/polagonow/pola/repository"

	"features-showcase/db/models"
)

// ProductRepository defines the contract for product persistence
// operations. The entity type is the canonical model in db/models — the
// single source of truth for the schema and the transport entity. It embeds the
// framework's standard CRUD contract; add custom query methods here.
type ProductRepository interface {
	repository.Repository[models.Product, uint]
}

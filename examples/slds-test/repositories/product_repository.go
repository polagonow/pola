package repositories

import (
	"github.com/polagonow/pola/repository"

	"slds-test/db/models"
)

// ProductRepository defines the contract for product persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type ProductRepository interface {
	repository.Repository[models.Product, uint]
}

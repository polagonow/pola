package repositories

import (
	"github.com/polagonow/pola/repository"
)

// Product represents the product entity for the repository layer.
type Product struct {
	ID     uint   `json:"id"`
	Name   string `json:"name" validate:"required"`
	Amount int    `json:"amount" validate:"required"`
}

// ProductRepository defines the contract for product persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type ProductRepository interface {
	repository.Repository[Product, uint]
}

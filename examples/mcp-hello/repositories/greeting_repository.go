package repositories

import (
	"github.com/polagonow/pola/repository"
)

// Greeting represents the greeting entity for the repository layer.
type Greeting struct {
	ID      uint   `json:"id"`
	Message string `json:"message" validate:"required"`
}

// GreetingRepository defines the contract for greeting persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type GreetingRepository interface {
	repository.Repository[Greeting, uint]
}

package repositories

import (
	"github.com/polagonow/pola/repository"

	"mcp-hello/db/models"
)

// GreetingRepository defines the contract for greeting persistence
// operations. It embeds the framework's standard CRUD contract; add custom
// query methods here.
type GreetingRepository interface {
	repository.Repository[models.Greeting, uint]
}

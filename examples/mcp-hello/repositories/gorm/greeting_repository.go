package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"mcp-hello/db/models"
	"mcp-hello/repositories"
)

// greetingRepository is the GORM-backed GreetingRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type greetingRepository struct {
	repository.Repository[models.Greeting, uint]
	db *gorm.DB
}

// NewGreetingRepository creates a new GORM-backed GreetingRepository, resolving
// its dependencies from the DI registry.
func NewGreetingRepository(r *core.Registry) repositories.GreetingRepository {
	db := core.MustInvoke[*gorm.DB](r)
	return &greetingRepository{
		Repository: gormrepo.New[models.Greeting, uint](db),
		db:         db,
	}
}

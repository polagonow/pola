package gorm

import (
	"gorm.io/gorm"

	"github.com/polagonow/pola/repository"
	gormrepo "github.com/polagonow/pola/repository/gorm"

	"mcp-hello/repositories"
)

// greetingRepository is the GORM-backed GreetingRepository. The
// embedded generic implementation supplies Create/Get/List/Update/Delete;
// add custom queries as methods on this struct using r.db.
type greetingRepository struct {
	repository.Repository[repositories.Greeting, uint]
	db *gorm.DB
}

// NewGreetingRepository creates a new GORM-backed GreetingRepository.
func NewGreetingRepository(db *gorm.DB) repositories.GreetingRepository {
	return &greetingRepository{
		Repository: gormrepo.New[repositories.Greeting, uint](db),
		db:         db,
	}
}

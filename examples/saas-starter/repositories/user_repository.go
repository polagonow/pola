package repositories

import (
	"context"

	"github.com/polagonow/pola/repository"

	"saas-starter/db/models"
)

// UserRepository defines the contract for user persistence operations.
type UserRepository interface {
	repository.Repository[models.User, uint]

	// GetByEmail returns the active (non-soft-deleted) user with the given email,
	// or an error if none exists.
	GetByEmail(ctx context.Context, email string) (*models.User, error)
}

package repositories

import (
	"context"

	"github.com/polagonow/pola/repository"

	"saas-starter/db/models"
)

// TeamRepository defines the contract for team persistence operations.
type TeamRepository interface {
	repository.Repository[models.Team, uint]

	// GetForUser returns the team the given user belongs to (via team_members).
	GetForUser(ctx context.Context, userID uint) (*models.Team, error)
	// GetByStripeCustomerID returns the team owning the given Stripe customer.
	GetByStripeCustomerID(ctx context.Context, customerID string) (*models.Team, error)
}

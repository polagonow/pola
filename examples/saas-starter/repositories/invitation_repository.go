package repositories

import (
	"context"

	"github.com/polagonow/pola/repository"

	"saas-starter/db/models"
)

// InvitationRepository defines the contract for invitation persistence operations.
type InvitationRepository interface {
	repository.Repository[models.Invitation, uint]

	// GetPending returns a pending invitation for (teamID, email), or an error if none.
	GetPending(ctx context.Context, teamID uint, email string) (*models.Invitation, error)
}

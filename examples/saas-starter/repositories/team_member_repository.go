package repositories

import (
	"context"

	"github.com/polagonow/pola/repository"

	"saas-starter/db/models"
)

// TeamMemberRepository defines the contract for team_member persistence operations.
type TeamMemberRepository interface {
	repository.Repository[models.TeamMember, uint]

	// GetByUserID returns the membership record for a user, or an error if none.
	GetByUserID(ctx context.Context, userID uint) (*models.TeamMember, error)
	// ListByTeam returns all memberships of a team.
	ListByTeam(ctx context.Context, teamID uint) ([]models.TeamMember, error)
}

package gorm

import (
	"context"

	"saas-starter/db/models"
)

// GetByUserID returns the membership record for a user.
func (r *teamMemberRepository) GetByUserID(ctx context.Context, userID uint) (*models.TeamMember, error) {
	var m models.TeamMember
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// ListByTeam returns all memberships of a team.
func (r *teamMemberRepository) ListByTeam(ctx context.Context, teamID uint) ([]models.TeamMember, error) {
	var ms []models.TeamMember
	if err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&ms).Error; err != nil {
		return nil, err
	}
	return ms, nil
}

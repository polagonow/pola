package gorm

import (
	"context"

	"saas-starter/db/models"
)

// GetPending returns a pending invitation for (teamID, email).
func (r *invitationRepository) GetPending(ctx context.Context, teamID uint, email string) (*models.Invitation, error) {
	var inv models.Invitation
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND email = ? AND status = ?", teamID, email, "pending").
		First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

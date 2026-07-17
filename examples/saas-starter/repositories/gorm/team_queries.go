package gorm

import (
	"context"

	"saas-starter/db/models"
)

// GetForUser returns the team the given user belongs to (via team_members).
func (r *teamRepository) GetForUser(ctx context.Context, userID uint) (*models.Team, error) {
	var t models.Team
	if err := r.db.WithContext(ctx).
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ?", userID).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByStripeCustomerID returns the team owning the given Stripe customer.
func (r *teamRepository) GetByStripeCustomerID(ctx context.Context, customerID string) (*models.Team, error) {
	var t models.Team
	if err := r.db.WithContext(ctx).
		Where("stripe_customer_id = ?", customerID).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

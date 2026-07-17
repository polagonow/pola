package gorm

import (
	"context"

	"saas-starter/db/models"
)

// GetByEmail returns the active (non-soft-deleted) user with the given email.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	if err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

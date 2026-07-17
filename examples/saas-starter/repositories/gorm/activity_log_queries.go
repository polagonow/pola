package gorm

import (
	"context"

	"saas-starter/db/models"
)

// ListForUser returns the most recent activity log entries for a user, newest
// first, capped at limit (limit <= 0 means unbounded).
func (r *activityLogRepository) ListForUser(ctx context.Context, userID uint, limit int) ([]models.ActivityLog, error) {
	var logs []models.ActivityLog
	q := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("timestamp DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

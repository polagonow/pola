package repositories

import (
	"context"

	"github.com/polagonow/pola/repository"

	"saas-starter/db/models"
)

// ActivityLogRepository defines the contract for activity_log persistence operations.
type ActivityLogRepository interface {
	repository.Repository[models.ActivityLog, uint]

	// ListForUser returns the most recent activity log entries for a user,
	// newest first, capped at limit.
	ListForUser(ctx context.Context, userID uint, limit int) ([]models.ActivityLog, error)
}

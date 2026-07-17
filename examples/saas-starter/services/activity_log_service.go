package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// ActivityLogServiceInterface is the contract for activity_log business logic.
// It embeds the framework's standard CRUD service; add custom business methods
// here. Routes and other call sites depend on this interface.
type ActivityLogServiceInterface interface {
	service.Service[models.ActivityLog, uint]
}

// ActivityLogService handles business logic for activity_log operations. The
// embedded generic service delegates CRUD to the repository; override a method
// (e.g. Create) on this struct to add validation or business rules, using
// s.repo. Add custom queries as methods here too.
type ActivityLogService struct {
	service.Service[models.ActivityLog, uint]
	repo repositories.ActivityLogRepository
}

// NewActivityLogService creates a new ActivityLogService, resolving its
// dependencies from the DI registry.
func NewActivityLogService(r *core.Registry) *ActivityLogService {
	repo := core.MustInvoke[repositories.ActivityLogRepository](r)
	return &ActivityLogService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ ActivityLogServiceInterface = (*ActivityLogService)(nil)

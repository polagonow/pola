package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// InvitationServiceInterface is the contract for invitation business logic.
// It embeds the framework's standard CRUD service; add custom business methods
// here. Routes and other call sites depend on this interface.
type InvitationServiceInterface interface {
	service.Service[models.Invitation, uint]
}

// InvitationService handles business logic for invitation operations. The
// embedded generic service delegates CRUD to the repository; override a method
// (e.g. Create) on this struct to add validation or business rules, using
// s.repo. Add custom queries as methods here too.
type InvitationService struct {
	service.Service[models.Invitation, uint]
	repo repositories.InvitationRepository
}

// NewInvitationService creates a new InvitationService, resolving its
// dependencies from the DI registry.
func NewInvitationService(r *core.Registry) *InvitationService {
	repo := core.MustInvoke[repositories.InvitationRepository](r)
	return &InvitationService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ InvitationServiceInterface = (*InvitationService)(nil)

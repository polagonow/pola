package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// TeamMemberServiceInterface is the contract for team_member business logic.
// It embeds the framework's standard CRUD service; add custom business methods
// here. Routes and other call sites depend on this interface.
type TeamMemberServiceInterface interface {
	service.Service[models.TeamMember, uint]
}

// TeamMemberService handles business logic for team_member operations. The
// embedded generic service delegates CRUD to the repository; override a method
// (e.g. Create) on this struct to add validation or business rules, using
// s.repo. Add custom queries as methods here too.
type TeamMemberService struct {
	service.Service[models.TeamMember, uint]
	repo repositories.TeamMemberRepository
}

// NewTeamMemberService creates a new TeamMemberService, resolving its
// dependencies from the DI registry.
func NewTeamMemberService(r *core.Registry) *TeamMemberService {
	repo := core.MustInvoke[repositories.TeamMemberRepository](r)
	return &TeamMemberService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ TeamMemberServiceInterface = (*TeamMemberService)(nil)

package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// TeamServiceInterface is the contract for team business logic.
// It embeds the framework's standard CRUD service; add custom business methods
// here. Routes and other call sites depend on this interface.
type TeamServiceInterface interface {
	service.Service[models.Team, uint]
}

// TeamService handles business logic for team operations. The
// embedded generic service delegates CRUD to the repository; override a method
// (e.g. Create) on this struct to add validation or business rules, using
// s.repo. Add custom queries as methods here too.
type TeamService struct {
	service.Service[models.Team, uint]
	repo repositories.TeamRepository
}

// NewTeamService creates a new TeamService, resolving its
// dependencies from the DI registry.
func NewTeamService(r *core.Registry) *TeamService {
	repo := core.MustInvoke[repositories.TeamRepository](r)
	return &TeamService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ TeamServiceInterface = (*TeamService)(nil)

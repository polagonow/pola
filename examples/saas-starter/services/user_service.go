package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"saas-starter/db/models"
	"saas-starter/repositories"
)

// UserServiceInterface is the contract for user business logic.
// It embeds the framework's standard CRUD service; add custom business methods
// here. Routes and other call sites depend on this interface.
type UserServiceInterface interface {
	service.Service[models.User, uint]
}

// UserService handles business logic for user operations. The
// embedded generic service delegates CRUD to the repository; override a method
// (e.g. Create) on this struct to add validation or business rules, using
// s.repo. Add custom queries as methods here too.
type UserService struct {
	service.Service[models.User, uint]
	repo repositories.UserRepository
}

// NewUserService creates a new UserService, resolving its
// dependencies from the DI registry.
func NewUserService(r *core.Registry) *UserService {
	repo := core.MustInvoke[repositories.UserRepository](r)
	return &UserService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ UserServiceInterface = (*UserService)(nil)

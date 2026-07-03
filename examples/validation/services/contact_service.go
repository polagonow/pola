package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"validation/repositories"
)

// ContactServiceInterface is the contract for contact business logic. It embeds
// the framework's standard CRUD service; add custom business methods here.
// Routes and other call sites depend on this interface.
type ContactServiceInterface interface {
	service.Service[repositories.Contact, uint]
}

// ContactService handles business logic for contact operations. The embedded
// generic service delegates CRUD to the repository; override a method (e.g.
// Create) on this struct to add validation or business rules, using s.repo.
type ContactService struct {
	service.Service[repositories.Contact, uint]
	repo repositories.ContactRepository
}

// NewContactService creates a new ContactService, resolving its dependencies
// from the DI registry.
func NewContactService(r *core.Registry) *ContactService {
	repo := core.MustInvoke[repositories.ContactRepository](r)
	return &ContactService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ ContactServiceInterface = (*ContactService)(nil)

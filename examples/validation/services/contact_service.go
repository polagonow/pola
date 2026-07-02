package services

import (
	"validation/repositories"

	"github.com/polagonow/pola/service"
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

// NewContactService creates a new ContactService.
func NewContactService(repo repositories.ContactRepository) *ContactService {
	return &ContactService{
		Service: service.New[repositories.Contact, uint](repo),
		repo:    repo,
	}
}

var _ ContactServiceInterface = (*ContactService)(nil)

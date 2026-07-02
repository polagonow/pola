package services

import (
	"mcp-hello/repositories"

	"github.com/polagonow/pola/service"
)

// GreetingServiceInterface is the contract for greeting business logic. It embeds
// the framework's standard CRUD service; add custom business methods here.
// Routes and other call sites depend on this interface.
type GreetingServiceInterface interface {
	service.Service[repositories.Greeting, uint]
}

// GreetingService handles business logic for greeting operations. The embedded
// generic service delegates CRUD to the repository; override a method (e.g.
// Create) on this struct to add validation or business rules, using s.repo.
type GreetingService struct {
	service.Service[repositories.Greeting, uint]
	repo repositories.GreetingRepository
}

// NewGreetingService creates a new GreetingService.
func NewGreetingService(repo repositories.GreetingRepository) *GreetingService {
	return &GreetingService{
		Service: service.New[repositories.Greeting, uint](repo),
		repo:    repo,
	}
}

var _ GreetingServiceInterface = (*GreetingService)(nil)

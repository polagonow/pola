package services

import (
	"context"

	"mcp-hello/repositories"

	"github.com/polagonow/pola/repository"
)

// GreetingService handles business logic for greeting operations.
type GreetingService struct {
	repo repositories.GreetingRepository
}

// NewGreetingService creates a new GreetingService.
func NewGreetingService(repo repositories.GreetingRepository) *GreetingService {
	return &GreetingService{repo: repo}
}

// Create creates a new greeting.
func (s *GreetingService) Create(ctx context.Context, entity *repositories.Greeting) error {
	// TODO: add business logic / validation
	return s.repo.Create(ctx, entity)
}

// Get returns a greeting by its ID.
func (s *GreetingService) Get(ctx context.Context, id uint) (*repositories.Greeting, error) {
	return s.repo.Get(ctx, id)
}

// List returns a paginated list of greetings.
func (s *GreetingService) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*repositories.Greeting], error) {
	return s.repo.List(ctx, params)
}

// Update updates an existing greeting.
func (s *GreetingService) Update(ctx context.Context, entity *repositories.Greeting) error {
	// TODO: add business logic / validation
	return s.repo.Update(ctx, entity)
}

// Delete removes a greeting by its ID.
func (s *GreetingService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

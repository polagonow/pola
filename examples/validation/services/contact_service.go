package services

import (
	"context"

	"validation/repositories"

	"github.com/polagonow/pola/repository"
)

// ContactService handles business logic for contact operations.
type ContactService struct {
	repo repositories.ContactRepository
}

// NewContactService creates a new ContactService.
func NewContactService(repo repositories.ContactRepository) *ContactService {
	return &ContactService{repo: repo}
}

func (s *ContactService) Create(ctx context.Context, entity *repositories.Contact) error {
	return s.repo.Create(ctx, entity)
}

func (s *ContactService) Get(ctx context.Context, id uint) (*repositories.Contact, error) {
	return s.repo.Get(ctx, id)
}

func (s *ContactService) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*repositories.Contact], error) {
	return s.repo.List(ctx, params)
}

func (s *ContactService) Update(ctx context.Context, entity *repositories.Contact) error {
	return s.repo.Update(ctx, entity)
}

func (s *ContactService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

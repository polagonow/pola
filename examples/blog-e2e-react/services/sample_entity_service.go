package services

import (
	"context"

	"blog-e2e-react/repositories"
)

// SampleEntityService handles business logic for sample_entity operations.
type SampleEntityService struct {
	repo repositories.SampleEntityRepository
}

// NewSampleEntityService creates a new SampleEntityService.
func NewSampleEntityService(repo repositories.SampleEntityRepository) *SampleEntityService {
	return &SampleEntityService{repo: repo}
}

// Create creates a new sample_entity.
func (s *SampleEntityService) Create(ctx context.Context, entity *repositories.SampleEntity) error {
	// TODO: add business logic / validation
	return s.repo.Create(ctx, entity)
}

// Get returns a sample_entity by its ID.
func (s *SampleEntityService) Get(ctx context.Context, id uint) (*repositories.SampleEntity, error) {
	return s.repo.Get(ctx, id)
}

// List returns a paginated list of sample_entities.
func (s *SampleEntityService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.SampleEntity], error) {
	return s.repo.List(ctx, params)
}

// Update updates an existing sample_entity.
func (s *SampleEntityService) Update(ctx context.Context, entity *repositories.SampleEntity) error {
	// TODO: add business logic / validation
	return s.repo.Update(ctx, entity)
}

// Delete removes a sample_entity by its ID.
func (s *SampleEntityService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

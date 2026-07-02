package services

import (
	"blog-e2e-react/repositories"

	"github.com/polagonow/pola/service"
)

// SampleEntityServiceInterface is the contract for sample_entity business logic. It embeds
// the framework's standard CRUD service; add custom business methods here.
// Routes and other call sites depend on this interface.
type SampleEntityServiceInterface interface {
	service.Service[repositories.SampleEntity, uint]
}

// SampleEntityService handles business logic for sample_entity operations. The embedded
// generic service delegates CRUD to the repository; override a method (e.g.
// Create) on this struct to add validation or business rules, using s.repo.
type SampleEntityService struct {
	service.Service[repositories.SampleEntity, uint]
	repo repositories.SampleEntityRepository
}

// NewSampleEntityService creates a new SampleEntityService.
func NewSampleEntityService(repo repositories.SampleEntityRepository) *SampleEntityService {
	return &SampleEntityService{
		Service: service.New[repositories.SampleEntity, uint](repo),
		repo:    repo,
	}
}

var _ SampleEntityServiceInterface = (*SampleEntityService)(nil)

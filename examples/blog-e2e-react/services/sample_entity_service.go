package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"blog-e2e-react/repositories"
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

// NewSampleEntityService creates a new SampleEntityService, resolving its
// dependencies from the DI registry.
func NewSampleEntityService(r *core.Registry) *SampleEntityService {
	repo := core.MustInvoke[repositories.SampleEntityRepository](r)
	return &SampleEntityService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ SampleEntityServiceInterface = (*SampleEntityService)(nil)

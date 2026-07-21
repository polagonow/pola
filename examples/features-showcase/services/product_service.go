package services

import (
	"github.com/polagonow/pola/core"
	"github.com/polagonow/pola/service"

	"features-showcase/db/models"
	"features-showcase/repositories"
)

// ProductServiceInterface is the contract for product business logic.
// It embeds the framework's standard CRUD service; add custom business methods
// here. Routes and other call sites depend on this interface.
type ProductServiceInterface interface {
	service.Service[models.Product, uint]
}

// ProductService handles business logic for product operations. The
// embedded generic service delegates CRUD to the repository; override a method
// (e.g. Create) on this struct to add validation or business rules, using
// s.repo. Add custom queries as methods here too.
type ProductService struct {
	service.Service[models.Product, uint]
	repo repositories.ProductRepository
}

// NewProductService creates a new ProductService, resolving its
// dependencies from the DI registry.
func NewProductService(r *core.Registry) *ProductService {
	repo := core.MustInvoke[repositories.ProductRepository](r)
	return &ProductService{
		Service: service.New(repo),
		repo:    repo,
	}
}

var _ ProductServiceInterface = (*ProductService)(nil)

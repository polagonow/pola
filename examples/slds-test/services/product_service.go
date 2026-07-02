package services

import (
	"slds-test/repositories"

	"github.com/polagonow/pola/service"
)

// ProductServiceInterface is the contract for product business logic. It embeds
// the framework's standard CRUD service; add custom business methods here.
// Routes and other call sites depend on this interface.
type ProductServiceInterface interface {
	service.Service[repositories.Product, uint]
}

// ProductService handles business logic for product operations. The embedded
// generic service delegates CRUD to the repository; override a method (e.g.
// Create) on this struct to add validation or business rules, using s.repo.
type ProductService struct {
	service.Service[repositories.Product, uint]
	repo repositories.ProductRepository
}

// NewProductService creates a new ProductService.
func NewProductService(repo repositories.ProductRepository) *ProductService {
	return &ProductService{
		Service: service.New[repositories.Product, uint](repo),
		repo:    repo,
	}
}

var _ ProductServiceInterface = (*ProductService)(nil)

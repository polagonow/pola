package services

import (
	"context"

	"my-app/repositories"
)

// ProductService handles business logic for product operations.
type ProductService struct {
	repo repositories.ProductRepository
}

// NewProductService creates a new ProductService.
func NewProductService(repo repositories.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// Create creates a new product.
func (s *ProductService) Create(ctx context.Context, entity *repositories.Product) error {
	// TODO: add business logic / validation
	return s.repo.Create(ctx, entity)
}

// Get returns a product by its ID.
func (s *ProductService) Get(ctx context.Context, id uint) (*repositories.Product, error) {
	return s.repo.Get(ctx, id)
}

// List returns a paginated list of products.
func (s *ProductService) List(ctx context.Context, params repositories.ListParams) (*repositories.ListResult[*repositories.Product], error) {
	return s.repo.List(ctx, params)
}

// Update updates an existing product.
func (s *ProductService) Update(ctx context.Context, entity *repositories.Product) error {
	// TODO: add business logic / validation
	return s.repo.Update(ctx, entity)
}

// Delete removes a product by its ID.
func (s *ProductService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

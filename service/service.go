// Package service defines the framework-neutral business-logic contract that
// generated per-entity services embed, plus a generic implementation that
// delegates to a repository.Repository. It is the service-layer twin of the
// repository package: one neutral CRUD contract, with a place (the generated
// wrapper) to add business rules by overriding methods.
package service

import (
	"context"

	"github.com/polagonow/pola/repository"
)

// Service is the standard CRUD-plus-pagination business-logic contract for
// entity T keyed by ID. It mirrors repository.Repository; generated per-entity
// service interfaces embed it so call sites keep a named, mockable, extensible
// interface (e.g. services.UserServiceInterface).
type Service[T any, ID comparable] interface {
	Create(ctx context.Context, entity *T) error
	Get(ctx context.Context, id ID) (*T, error)
	List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*T], error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id ID) error
}

// New returns a Service backed by repo. Every method delegates straight to the
// repository; to add business logic or validation, override the corresponding
// method on the generated service struct (which embeds this) and call through
// to its retained repository field.
func New[T any, ID comparable](repo repository.Repository[T, ID]) Service[T, ID] {
	return &service[T, ID]{repo: repo}
}

type service[T any, ID comparable] struct {
	repo repository.Repository[T, ID]
}

func (s *service[T, ID]) Create(ctx context.Context, entity *T) error {
	return s.repo.Create(ctx, entity)
}

func (s *service[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	return s.repo.Get(ctx, id)
}

func (s *service[T, ID]) List(ctx context.Context, params repository.ListParams) (*repository.ListResult[*T], error) {
	return s.repo.List(ctx, params)
}

func (s *service[T, ID]) Update(ctx context.Context, entity *T) error {
	return s.repo.Update(ctx, entity)
}

func (s *service[T, ID]) Delete(ctx context.Context, id ID) error {
	return s.repo.Delete(ctx, id)
}

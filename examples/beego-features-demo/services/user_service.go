package services

import (
	"context"
	"errors"

	"github.com/polagonow/pola/auth/password"
	"github.com/polagonow/pola/core"
	"beego-features-demo/repositories"
)

type UserService struct {
	repo repositories.UserRepository
}

func NewUserService(r *core.Registry) *UserService {
	return &UserService{repo: core.MustInvoke[repositories.UserRepository](r)}
}

func (s *UserService) Register(ctx context.Context, username, pass, displayName string) (*repositories.User, error) {
	hash, err := password.Hash(pass)
	if err != nil {
		return nil, err
	}
	user := &repositories.User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  displayName,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Authenticate(ctx context.Context, username, pass string) (*repositories.User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	ok, err := password.Verify(pass, user.PasswordHash)
	if err != nil || !ok {
		return nil, errors.New("invalid credentials")
	}
	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*repositories.User, error) {
	return s.repo.GetByID(ctx, id)
}

package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/polagonow/pola/core"
	"beego-features-demo/db/models"
	"beego-features-demo/repositories"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(r *core.Registry) repositories.UserRepository {
	return &userRepository{db: core.MustInvoke[*gorm.DB](r)}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}

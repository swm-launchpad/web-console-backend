package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, userID uint) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Delete(ctx context.Context, userID uint) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

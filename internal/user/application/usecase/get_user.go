package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type GetUserInput struct {
	UserID uint
}

type GetUserOutput struct {
	UserID       uint
	Username     string
	Email        string
	Name         string
	Phone        string
	Organization string
	Status       string
	CreatedAt    time.Time
}

type GetUserUseCase struct {
	userRepo repository.UserRepository
}

func NewGetUserUseCase(userRepo repository.UserRepository) *GetUserUseCase {
	return &GetUserUseCase{
		userRepo: userRepo,
	}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	// Validate input
	if input.UserID == 0 {
		return nil, errors.New("user ID is required")
	}

	// Find user by ID
	user, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Map domain model to output
	output := &GetUserOutput{
		UserID:    user.UserID,
		Username:  user.Username,
		Email:     user.Email,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	}

	// Map optional fields
	if user.Name != nil {
		output.Name = *user.Name
	}
	if user.Phone != nil {
		output.Phone = *user.Phone
	}
	if user.Organization != nil {
		output.Organization = *user.Organization
	}

	return output, nil
}

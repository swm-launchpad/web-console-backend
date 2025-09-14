package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

type GetUserInput struct {
	UserID string
}

type GetUserOutput struct {
	UserID          string
	Username        string
	Email           string
	Name            string
	ProfileImageURL string
	Department      string
	Role            string
	Status          string
	LastLoginAt     *time.Time
	CreatedAt       time.Time
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
	if input.UserID == "" {
		return nil, errors.New("user ID is required")
	}

	// Find user by ID
	user, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Map domain model to output
	output := &GetUserOutput{
		UserID:      user.UserID,
		Username:    user.Username,
		Status:      string(user.Status),
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
	}

	// Map optional fields
	if user.Email != nil {
		output.Email = *user.Email
	}
	if user.Name != nil {
		output.Name = *user.Name
	}
	if user.ProfileImageURL != nil {
		output.ProfileImageURL = *user.ProfileImageURL
	}
	if user.Department != nil {
		output.Department = *user.Department
	}
	if user.Role != nil {
		output.Role = *user.Role
	}

	return output, nil
}
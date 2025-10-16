package application

import (
	"context"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type ChangePasswordInput struct {
	UserID          uint
	CurrentPassword string
	NewPassword     string
}

type ChangePasswordOutput struct {
	Success bool
}

type ChangePasswordUseCase struct {
	userService service.UserService
	authService service.AuthService
}

func NewChangePasswordUseCase(
	userService service.UserService,
	authService service.AuthService,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		userService: userService,
		authService: authService,
	}
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) (*ChangePasswordOutput, error) {
	// Get user
	user, err := uc.userService.GetUserByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Validate user is active
	if err := uc.userService.ValidateUserCredentials(ctx, user); err != nil {
		return nil, err
	}

	// Verify current password
	if err := uc.authService.VerifyPassword(user.PasswordHash, input.CurrentPassword); err != nil {
		return nil, usererrors.ErrInvalidCredentials
	}

	// Validate new password (minimum 8 characters)
	if len(input.NewPassword) < 8 {
		return nil, usererrors.ErrWeakPassword
	}

	// Hash new password
	newPasswordHash, err := uc.authService.HashPassword(input.NewPassword)
	if err != nil {
		return nil, err
	}

	// Update password
	if err := uc.userService.UpdatePassword(ctx, input.UserID, newPasswordHash); err != nil {
		return nil, err
	}

	return &ChangePasswordOutput{Success: true}, nil
}

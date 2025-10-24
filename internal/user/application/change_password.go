package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
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
	logger      logger.Logger
}

func NewChangePasswordUseCase(
	userService service.UserService,
	authService service.AuthService,
	log logger.Logger,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		userService: userService,
		authService: authService,
		logger:      log,
	}
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, input ChangePasswordInput) (*ChangePasswordOutput, error) {
	uc.logger.Info(ctx, "change password started",
		zap.Uint("user_id", input.UserID),
	)

	// Get user
	user, err := uc.userService.GetUserByID(ctx, input.UserID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get user",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	// Validate user is active
	if err := uc.userService.ValidateUserCredentials(ctx, user); err != nil {
		uc.logger.Error(ctx, "user credentials validation failed",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	// Verify current password
	if err := uc.authService.VerifyPassword(user.PasswordHash, input.CurrentPassword); err != nil {
		uc.logger.Error(ctx, "current password verification failed",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, usererrors.ErrInvalidCredentials
	}

	// Validate new password (minimum 8 characters)
	if len(input.NewPassword) < 8 {
		uc.logger.Warn(ctx, "new password too weak",
			zap.Uint("user_id", input.UserID),
			zap.Int("password_length", len(input.NewPassword)),
		)
		return nil, usererrors.ErrWeakPassword
	}

	// Hash new password
	newPasswordHash, err := uc.authService.HashPassword(input.NewPassword)
	if err != nil {
		uc.logger.Error(ctx, "failed to hash new password",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	// Update password
	if err := uc.userService.UpdatePassword(ctx, input.UserID, newPasswordHash); err != nil {
		uc.logger.Error(ctx, "failed to update password",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "change password completed",
		zap.Uint("user_id", input.UserID),
	)

	return &ChangePasswordOutput{Success: true}, nil
}

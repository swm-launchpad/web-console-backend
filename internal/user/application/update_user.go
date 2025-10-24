package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type UpdateUserInput struct {
	UserID       uint
	Name         *string
	Phone        *string
	Organization *string
}

type UpdateUserOutput struct {
	UserID       uint
	Username     string
	Email        string
	Name         string
	Phone        string
	Organization string
	Status       string
}

type UpdateUserUseCase struct {
	userService service.UserService
	logger      logger.Logger
}

func NewUpdateUserUseCase(userService service.UserService, log logger.Logger) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userService: userService,
		logger:      log,
	}
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (*UpdateUserOutput, error) {
	uc.logger.Info(ctx, "update user started",
		zap.Uint("user_id", input.UserID),
	)

	// Get existing user
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

	// Update fields
	updated := false
	if input.Name != nil {
		user.Name = input.Name
		updated = true
	}
	if input.Phone != nil {
		user.Phone = input.Phone
		updated = true
	}
	if input.Organization != nil {
		user.Organization = input.Organization
		updated = true
	}

	// Save if any field was updated
	if !updated {
		uc.logger.Warn(ctx, "no fields to update",
			zap.Uint("user_id", input.UserID),
		)
		return nil, usererrors.ErrNoFieldsToUpdate
	}

	if err := uc.userService.UpdateUser(ctx, user); err != nil {
		uc.logger.Error(ctx, "failed to update user",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	// Map to output
	output := &UpdateUserOutput{
		UserID:   user.UserID,
		Username: user.Username,
		Email:    user.Email,
		Status:   string(user.Status),
	}

	if user.Name != nil {
		output.Name = *user.Name
	}
	if user.Phone != nil {
		output.Phone = *user.Phone
	}
	if user.Organization != nil {
		output.Organization = *user.Organization
	}

	uc.logger.Info(ctx, "update user completed",
		zap.Uint("user_id", user.UserID),
		zap.String("username", user.Username),
	)

	return output, nil
}

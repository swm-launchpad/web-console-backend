package application

import (
	"context"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
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
}

func NewUpdateUserUseCase(userService service.UserService) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userService: userService,
	}
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (*UpdateUserOutput, error) {
	// Get existing user
	user, err := uc.userService.GetUserByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Validate user is active
	if err := uc.userService.ValidateUserCredentials(ctx, user); err != nil {
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
		return nil, usererrors.ErrNoFieldsToUpdate
	}

	if err := uc.userService.UpdateUser(ctx, user); err != nil {
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

	return output, nil
}

package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
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
	userService service.UserService
}

func NewGetUserUseCase(userService service.UserService) *GetUserUseCase {
	return &GetUserUseCase{
		userService: userService,
	}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	// Get user through UserService
	user, err := uc.userService.GetUserByID(ctx, input.UserID)
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

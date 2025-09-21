package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type LoginUserInput struct {
	Username string
	Password string
}

type LoginUserOutput struct {
	UserID   uint
	Token    string
	Username string
	Email    string
	Name     string
}

type LoginUserUseCase struct {
	authService service.AuthService
}

func NewLoginUserUseCase(authService service.AuthService) *LoginUserUseCase {
	return &LoginUserUseCase{
		authService: authService,
	}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error) {
	// Authenticate user through AuthenticationService
	user, token, err := uc.authService.AuthenticateUser(ctx, input.Username, input.Password)
	if err != nil {
		return nil, err
	}

	// Prepare output
	output := &LoginUserOutput{
		UserID:   user.UserID,
		Token:    token,
		Username: user.Username,
		Email:    user.Email,
	}

	if user.Name != nil {
		output.Name = *user.Name
	}

	return output, nil
}

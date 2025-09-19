package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type RegisterUserInput struct {
	Username string
	Password string
	Email    string
	Name     string
}

type RegisterUserOutput struct {
	UserID uint
	Token  string
}

type RegisterUserUseCase struct {
	authService service.AuthService
}

func NewRegisterUserUseCase(authService service.AuthService) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		authService: authService,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	var name *string
	if input.Name != "" {
		name = &input.Name
	}

	// Register user through AuthenticationService
	user, token, err := uc.authService.RegisterUser(ctx, input.Username, input.Password, input.Email, name)
	if err != nil {
		return nil, err
	}

	return &RegisterUserOutput{
		UserID: user.UserID,
		Token:  token,
	}, nil
}

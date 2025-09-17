package usecase

import (
	"context"
	"errors"

	authErrors "github.com/swm-launchpad/web-console-backend/internal/common/auth/errors"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/service"
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
	userRepo    repository.UserRepository
	authService service.AuthService
}

func NewLoginUserUseCase(
	userRepo repository.UserRepository,
	authService service.AuthService,
) *LoginUserUseCase {
	return &LoginUserUseCase{
		userRepo:    userRepo,
		authService: authService,
	}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error) {
	// Validate input
	if input.Username == "" {
		return nil, errors.New("username is required")
	}
	if input.Password == "" {
		return nil, errors.New("password is required")
	}

	// Find user by username
	user, err := uc.userRepo.FindByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, authErrors.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, authErrors.ErrUserNotActive
	}

	// Verify password
	if err := uc.authService.VerifyPassword(user.PasswordHash, input.Password); err != nil {
		return nil, authErrors.ErrInvalidCredentials
	}

	// Generate JWT token
	token, err := uc.authService.GenerateToken(ctx, user.UserID)
	if err != nil {
		return nil, authErrors.ErrTokenGenerationFailed
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
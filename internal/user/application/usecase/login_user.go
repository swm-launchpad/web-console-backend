package usecase

import (
	"context"
	"errors"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
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
	userRepo     repository.UserRepository
	jwtUtil      *jwt.JWTUtil
	passwordUtil *password.PasswordUtil
}

func NewLoginUserUseCase(
	userRepo repository.UserRepository,
	jwtUtil *jwt.JWTUtil,
	passwordUtil *password.PasswordUtil,
) *LoginUserUseCase {
	return &LoginUserUseCase{
		userRepo:     userRepo,
		jwtUtil:      jwtUtil,
		passwordUtil: passwordUtil,
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
			return nil, auth.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, auth.ErrUserNotActive
	}

	// Verify password
	if err := uc.passwordUtil.VerifyPassword(user.PasswordHash, input.Password); err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	// Generate JWT token
	token, err := uc.jwtUtil.GenerateToken(ctx, user.UserID)
	if err != nil {
		return nil, auth.ErrTokenGenerationFailed
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

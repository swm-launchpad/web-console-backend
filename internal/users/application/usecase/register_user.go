package usecase

import (
	"context"
	"errors"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
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
	userRepo     repository.UserRepository
	jwtUtil      *jwt.JWTUtil
	passwordUtil *password.PasswordUtil
}

func NewRegisterUserUseCase(
	userRepo repository.UserRepository,
	jwtUtil *jwt.JWTUtil,
	passwordUtil *password.PasswordUtil,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		userRepo:     userRepo,
		jwtUtil:      jwtUtil,
		passwordUtil: passwordUtil,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, err
	}

	// Check if username already exists
	exists, err := uc.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	exists, err = uc.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already exists")
	}

	// Create new user
	user, err := model.NewUser(input.Username, input.Email)
	if err != nil {
		return nil, err
	}

	// Set additional fields
	if input.Name != "" {
		user.Name = &input.Name
	}

	// Hash password
	hashedPassword, err := uc.passwordUtil.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	user.UpdatePassword(hashedPassword)

	// Activate user immediately (no email verification for now)
	if err := user.Activate(); err != nil {
		return nil, err
	}

	// Save user to repository
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := uc.jwtUtil.GenerateToken(ctx, user.UserID)
	if err != nil {
		return nil, auth.ErrTokenGenerationFailed
	}

	return &RegisterUserOutput{
		UserID: user.UserID,
		Token:  token,
	}, nil
}

func (uc *RegisterUserUseCase) validateInput(input RegisterUserInput) error {
	if input.Username == "" {
		return errors.New("username is required")
	}
	if len(input.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	if input.Password == "" {
		return errors.New("password is required")
	}
	if len(input.Password) < 8 {
		return auth.ErrPasswordTooWeak
	}
	if input.Email == "" {
		return errors.New("email is required")
	}
	// Basic email validation
	if !contains(input.Email, "@") || !contains(input.Email, ".") {
		return errors.New("invalid email format")
	}
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
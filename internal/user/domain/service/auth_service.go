package service

import (
	"context"
	"errors"
	"strings"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	domainerrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/error"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

var (
	ErrInvalidCredentials = domainerrors.ErrInvalidCredentials
	ErrWeakPassword       = domainerrors.ErrWeakPassword
	ErrInvalidEmail       = domainerrors.ErrInvalidEmail
)

// AuthService defines the interface for authentication-related business logic
type AuthService interface {
	// RegisterUser registers a new user with validation
	RegisterUser(ctx context.Context, username, plainPassword, email string, name *string) (*model.User, string, error)

	// AuthenticateUser authenticates a user and returns a token
	AuthenticateUser(ctx context.Context, username, plainPassword string) (*model.User, string, error)

	// ValidateRegistrationInput validates user registration input
	ValidateRegistrationInput(username, plainPassword, email string) error

	// ValidateLoginInput validates user login input
	ValidateLoginInput(username, plainPassword string) error

	// GenerateToken generates a JWT token for the user
	GenerateToken(ctx context.Context, userID uint) (string, error)

	// HashPassword hashes a plain text password
	HashPassword(plainPassword string) (string, error)

	// VerifyPassword verifies a plain text password against a hash
	VerifyPassword(passwordHash, plainPassword string) error
}

// authService is the concrete implementation of AuthService
type authService struct {
	userService  UserService
	jwtUtil      *jwt.JWTUtil
	passwordUtil *password.PasswordUtil
}

// NewAuthService creates a new instance of AuthService
func NewAuthService(
	userService UserService,
	jwtUtil *jwt.JWTUtil,
	passwordUtil *password.PasswordUtil,
) AuthService {
	return &authService{
		userService:  userService,
		jwtUtil:      jwtUtil,
		passwordUtil: passwordUtil,
	}
}

// RegisterUser registers a new user with all necessary validations and setup
func (s *authService) RegisterUser(ctx context.Context, username, plainPassword, email string, name *string) (*model.User, string, error) {
	// Validate input
	if err := s.ValidateRegistrationInput(username, plainPassword, email); err != nil {
		return nil, "", err
	}

	// Check username availability
	if err := s.userService.CheckUsernameAvailability(ctx, username); err != nil {
		return nil, "", err
	}

	// Check email availability
	if err := s.userService.CheckEmailAvailability(ctx, email); err != nil {
		return nil, "", err
	}

	// Hash password
	passwordHash, err := s.HashPassword(plainPassword)
	if err != nil {
		return nil, "", err
	}

	// Create user through UserService
	user, err := s.userService.CreateUser(ctx, username, email, passwordHash, name)
	if err != nil {
		return nil, "", err
	}

	// Generate token
	token, err := s.GenerateToken(ctx, user.UserID)
	if err != nil {
		return nil, "", auth.ErrTokenGenerationFailed
	}

	return user, token, nil
}

// AuthenticateUser authenticates a user and returns a token
func (s *authService) AuthenticateUser(ctx context.Context, username, plainPassword string) (*model.User, string, error) {
	// Validate input
	if err := s.ValidateLoginInput(username, plainPassword); err != nil {
		return nil, "", err
	}

	// Get user by username
	user, err := s.userService.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Validate user credentials (check if active)
	if err := s.userService.ValidateUserCredentials(ctx, user); err != nil {
		if errors.Is(err, ErrUserNotActive) {
			return nil, "", auth.ErrUserNotActive
		}
		return nil, "", err
	}

	// Verify password
	if err := s.VerifyPassword(user.PasswordHash, plainPassword); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Generate token
	token, err := s.GenerateToken(ctx, user.UserID)
	if err != nil {
		return nil, "", auth.ErrTokenGenerationFailed
	}

	return user, token, nil
}

// ValidateRegistrationInput validates user registration input
func (s *authService) ValidateRegistrationInput(username, plainPassword, email string) error {
	// Validate username
	if username == "" {
		return domainerrors.ErrUsernameRequired
	}
	if len(username) < 3 {
		return domainerrors.ErrUsernameTooShort
	}

	// Validate password
	if plainPassword == "" {
		return domainerrors.ErrPasswordRequired
	}
	if len(plainPassword) < 8 {
		return ErrWeakPassword
	}

	// Validate email
	if email == "" {
		return domainerrors.ErrEmailRequired
	}
	// Basic email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return ErrInvalidEmail
	}

	return nil
}

// ValidateLoginInput validates user login input
func (s *authService) ValidateLoginInput(username, plainPassword string) error {
	if username == "" {
		return domainerrors.ErrUsernameRequired
	}
	if plainPassword == "" {
		return domainerrors.ErrPasswordRequired
	}
	return nil
}

// GenerateToken generates a JWT token for the user
func (s *authService) GenerateToken(ctx context.Context, userID uint) (string, error) {
	if userID == 0 {
		return "", domainerrors.ErrInvalidUserID
	}
	return s.jwtUtil.GenerateToken(ctx, userID)
}

// HashPassword hashes a plain text password
func (s *authService) HashPassword(plainPassword string) (string, error) {
	if plainPassword == "" {
		return "", domainerrors.ErrPasswordEmpty
	}
	return s.passwordUtil.HashPassword(plainPassword)
}

// VerifyPassword verifies a plain text password against a hash
func (s *authService) VerifyPassword(passwordHash, plainPassword string) error {
	if passwordHash == "" || plainPassword == "" {
		return ErrInvalidCredentials
	}
	return s.passwordUtil.VerifyPassword(passwordHash, plainPassword)
}

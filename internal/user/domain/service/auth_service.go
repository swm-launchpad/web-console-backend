package service

import (
	"context"
	"strings"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials = usererrors.ErrInvalidCredentials
	ErrWeakPassword       = usererrors.ErrWeakPassword
	ErrInvalidEmail       = usererrors.ErrInvalidEmail
)

// AuthService defines the interface for authentication-related business logic
type AuthService interface {
	// RegisterUser registers a new user with validation
	RegisterUser(ctx context.Context, email, plainPassword, nickname string) (*model.User, string, error)

	// AuthenticateUser authenticates a user and returns a token
	AuthenticateUser(ctx context.Context, email, plainPassword string) (*model.User, string, error)

	// ValidateRegistrationInput validates user registration input
	ValidateRegistrationInput(email, plainPassword, nickname string) error

	// ValidateLoginInput validates user login input
	ValidateLoginInput(email, plainPassword string) error

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
	logger       logger.Logger
}

// NewAuthService creates a new instance of AuthService
func NewAuthService(
	userService UserService,
	jwtUtil *jwt.JWTUtil,
	passwordUtil *password.PasswordUtil,
	log logger.Logger,
) AuthService {
	return &authService{
		userService:  userService,
		jwtUtil:      jwtUtil,
		passwordUtil: passwordUtil,
		logger:       log,
	}
}

// RegisterUser registers a new user with all necessary validations and setup
func (s *authService) RegisterUser(ctx context.Context, email, plainPassword, nickname string) (*model.User, string, error) {
	s.logger.Info(ctx, "register user started",
		zap.String("email", email),
		zap.String("nickname", nickname),
	)

	// Validate input
	if err := s.ValidateRegistrationInput(email, plainPassword, nickname); err != nil {
		s.logger.Error(ctx, "registration input validation failed",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, "", err
	}

	// Check email availability
	if err := s.userService.CheckEmailAvailability(ctx, email); err != nil {
		s.logger.Error(ctx, "email availability check failed",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, "", err
	}

	// Hash password
	passwordHash, err := s.HashPassword(plainPassword)
	if err != nil {
		s.logger.Error(ctx, "password hashing failed",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, "", err
	}

	// Create user through UserService
	user, err := s.userService.CreateUser(ctx, email, passwordHash, nickname)
	if err != nil {
		s.logger.Error(ctx, "user creation failed",
			zap.Error(err),
			zap.String("email", email),
			zap.String("nickname", nickname),
		)
		return nil, "", err
	}

	// Generate token
	token, err := s.GenerateToken(ctx, user.UserID)
	if err != nil {
		s.logger.Error(ctx, "token generation failed",
			zap.Error(err),
			zap.Uint("user_id", user.UserID),
		)
		return nil, "", usererrors.ErrTokenGenerationFailed
	}

	s.logger.Info(ctx, "register user completed",
		zap.Uint("user_id", user.UserID),
		zap.String("email", email),
	)

	return user, token, nil
}

// AuthenticateUser authenticates a user and returns a token
func (s *authService) AuthenticateUser(ctx context.Context, email, plainPassword string) (*model.User, string, error) {
	s.logger.Info(ctx, "authenticate user started",
		zap.String("email", email),
	)

	// Validate input
	if err := s.ValidateLoginInput(email, plainPassword); err != nil {
		s.logger.Error(ctx, "login input validation failed",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, "", err
	}

	// Get user by email
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Error(ctx, "user not found",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, "", ErrInvalidCredentials
	}

	// Validate user credentials (check if active)
	if err := s.userService.ValidateUserCredentials(ctx, user); err != nil {
		s.logger.Error(ctx, "user credentials validation failed",
			zap.Error(err),
			zap.Uint("user_id", user.UserID),
		)
		return nil, "", err
	}

	// Verify password
	if err := s.VerifyPassword(user.PasswordHash, plainPassword); err != nil {
		s.logger.Error(ctx, "password verification failed",
			zap.Error(err),
			zap.Uint("user_id", user.UserID),
		)
		return nil, "", ErrInvalidCredentials
	}

	// Generate token
	token, err := s.GenerateToken(ctx, user.UserID)
	if err != nil {
		s.logger.Error(ctx, "token generation failed",
			zap.Error(err),
			zap.Uint("user_id", user.UserID),
		)
		return nil, "", usererrors.ErrTokenGenerationFailed
	}

	s.logger.Info(ctx, "authenticate user completed",
		zap.Uint("user_id", user.UserID),
		zap.String("email", email),
	)

	return user, token, nil
}

// ValidateRegistrationInput validates user registration input
func (s *authService) ValidateRegistrationInput(email, plainPassword, nickname string) error {
	// Validate email
	if email == "" {
		return usererrors.ErrEmailRequired
	}
	// Basic email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return ErrInvalidEmail
	}

	// Validate password
	if plainPassword == "" {
		return usererrors.ErrPasswordRequired
	}
	if len(plainPassword) < 8 {
		return ErrWeakPassword
	}

	// Validate nickname
	if nickname == "" {
		return usererrors.ErrNicknameRequired
	}
	if len(nickname) < 2 {
		return usererrors.ErrNicknameTooShort
	}

	return nil
}

// ValidateLoginInput validates user login input
func (s *authService) ValidateLoginInput(email, plainPassword string) error {
	if email == "" {
		return usererrors.ErrEmailRequired
	}
	if plainPassword == "" {
		return usererrors.ErrPasswordRequired
	}
	return nil
}

// GenerateToken generates a JWT token for the user
func (s *authService) GenerateToken(ctx context.Context, userID uint) (string, error) {
	if userID == 0 {
		return "", usererrors.ErrInvalidUserID
	}
	return s.jwtUtil.GenerateToken(ctx, userID)
}

// HashPassword hashes a plain text password
func (s *authService) HashPassword(plainPassword string) (string, error) {
	if plainPassword == "" {
		return "", usererrors.ErrPasswordEmpty
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

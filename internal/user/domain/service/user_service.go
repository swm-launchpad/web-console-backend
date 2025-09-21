package service

import (
	"context"
	"time"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

var (
	ErrInvalidUserData = usererrors.ErrInvalidUserData
	ErrUserNotActive   = usererrors.ErrUserNotActive
)

// UserService defines the interface for user-related business logic
type UserService interface {
	// CreateUser creates a new user with the given information
	CreateUser(ctx context.Context, username, email string, passwordHash string, name *string) (*model.User, error)

	// GetUserByID retrieves a user by their ID
	GetUserByID(ctx context.Context, userID uint) (*model.User, error)

	// GetUserByUsername retrieves a user by their username
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)

	// GetUserByEmail retrieves a user by their email
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)

	// UpdateUser updates user information
	UpdateUser(ctx context.Context, user *model.User) error

	// ActivateUser activates a user account
	ActivateUser(ctx context.Context, userID uint) error

	// ValidateUserCredentials validates if user is active and can login
	ValidateUserCredentials(ctx context.Context, user *model.User) error

	// UpdatePassword updates user's password
	UpdatePassword(ctx context.Context, userID uint, passwordHash string) error

	// CheckUsernameAvailability checks if username is available
	CheckUsernameAvailability(ctx context.Context, username string) error

	// CheckEmailAvailability checks if email is available
	CheckEmailAvailability(ctx context.Context, email string) error
}

// userService is the concrete implementation of UserService
type userService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

// CreateUser creates a new user with validation
func (s *userService) CreateUser(ctx context.Context, username, email string, passwordHash string, name *string) (*model.User, error) {
	// Create user model
	user, err := model.NewUser(username, email)
	if err != nil {
		return nil, err
	}

	// Set additional fields
	if name != nil && *name != "" {
		user.Name = name
	}

	// Set password
	user.UpdatePassword(passwordHash)

	// Activate user immediately (no email verification for now)
	if err := user.Activate(); err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	if userID == 0 {
		return nil, ErrInvalidUserData
	}

	return s.userRepo.FindByID(ctx, userID)
}

// GetUserByUsername retrieves a user by username
func (s *userService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, ErrInvalidUserData
	}

	return s.userRepo.FindByUsername(ctx, username)
}

// GetUserByEmail retrieves a user by email
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, ErrInvalidUserData
	}

	return s.userRepo.FindByEmail(ctx, email)
}

// UpdateUser updates user information
func (s *userService) UpdateUser(ctx context.Context, user *model.User) error {
	if user == nil || user.UserID == 0 {
		return ErrInvalidUserData
	}

	// Update timestamp
	now := time.Now()
	user.UpdatedAt = &now

	return s.userRepo.Update(ctx, user)
}

// ActivateUser activates a user account
func (s *userService) ActivateUser(ctx context.Context, userID uint) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := user.Activate(); err != nil {
		return err
	}

	return s.UpdateUser(ctx, user)
}

// ValidateUserCredentials validates if user is active and can login
func (s *userService) ValidateUserCredentials(ctx context.Context, user *model.User) error {
	if user == nil {
		return ErrInvalidUserData
	}

	if !user.IsActive() {
		return ErrUserNotActive
	}

	return nil
}

// UpdatePassword updates user's password
func (s *userService) UpdatePassword(ctx context.Context, userID uint, passwordHash string) error {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.UpdatePassword(passwordHash)

	return s.UpdateUser(ctx, user)
}

// CheckUsernameAvailability checks if username is available
func (s *userService) CheckUsernameAvailability(ctx context.Context, username string) error {
	if username == "" {
		return ErrInvalidUserData
	}

	exists, err := s.userRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return err
	}

	if exists {
		return usererrors.ErrUsernameExists
	}

	return nil
}

// CheckEmailAvailability checks if email is available
func (s *userService) CheckEmailAvailability(ctx context.Context, email string) error {
	if email == "" {
		return ErrInvalidUserData
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}

	if exists {
		return usererrors.ErrEmailExists
	}

	return nil
}

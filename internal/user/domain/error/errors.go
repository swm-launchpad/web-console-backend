// Package error defines domain-specific errors for the user bounded context.
// These errors are used across all layers to maintain consistent error handling.
package error

import "errors"

// Repository errors - errors that occur at the data access layer
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// Domain errors - business logic violations and domain rule errors
var (
	ErrInvalidUserData    = errors.New("invalid user data")
	ErrUserNotActive      = errors.New("user is not active")
	ErrCannotActivateUser = errors.New("cannot activate user")
	ErrCannotDeleteUser   = errors.New("cannot activate deleted user") // TODO: Fix message - should be "cannot activate deleted user"
)

// Authentication errors - errors related to user authentication and security
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("password is too weak")
	ErrInvalidEmail       = errors.New("invalid email format")
)

// Validation errors - input validation and data integrity errors
var (
	ErrUsernameRequired  = errors.New("username is required")
	ErrPasswordRequired  = errors.New("password is required")
	ErrEmailRequired     = errors.New("email is required")
	ErrUsernameTooShort  = errors.New("username must be at least 3 characters long")
	ErrInvalidUserID     = errors.New("invalid user ID")
	ErrPasswordEmpty     = errors.New("password cannot be empty")
)

// Duplicate errors - errors for unique constraint violations
var (
	ErrUsernameExists = errors.New("username already exists")
	ErrEmailExists    = errors.New("email already exists")
)
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
	ErrInvalidUserData           = errors.New("invalid user data")
	ErrUserNotActive             = errors.New("user is not active")
	ErrCannotActivateDeletedUser = errors.New("cannot activate deleted user")
	ErrCannotDeleteUser          = errors.New("cannot delete user")
)

// Authentication errors - errors related to user authentication and security
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("password is too weak")
	ErrInvalidEmail       = errors.New("invalid email format")
)

// Validation errors - input validation and data integrity errors
var (
	ErrUsernameRequired = errors.New("username is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrEmailRequired    = errors.New("email is required")
	ErrUsernameTooShort = errors.New("username must be at least 3 characters long")
	ErrInvalidUserID    = errors.New("invalid user ID")
	ErrPasswordEmpty    = errors.New("password cannot be empty")
)

// Duplicate errors - errors for unique constraint violations
var (
	ErrUsernameExists = errors.New("username already exists")
	ErrEmailExists    = errors.New("email already exists")
)

// Error codes for user domain
const (
	// User errors (USER_XXX)
	CodeUserNotFound              = "USER_001"
	CodeUserAlreadyExists         = "USER_002"
	CodeInvalidUserID             = "USER_003"
	CodeUserNotActive             = "USER_004"
	CodeInvalidUserData           = "USER_005"
	CodeCannotActivateDeletedUser = "USER_006"
	CodeCannotDeleteUser          = "USER_007"

	// Authentication errors - removed (use common/auth codes instead)
	CodeInvalidCredentials = "UAUTH_001" // User-specific auth error
	CodeWeakPassword       = "UAUTH_002" // User-specific auth error

	// Validation errors (UVAL_XXX) - user domain specific
	CodeUsernameRequired = "UVAL_001"
	CodePasswordRequired = "UVAL_002"
	CodeEmailRequired    = "UVAL_003"
	CodeUsernameTooShort = "UVAL_004"
	CodeInvalidEmail     = "UVAL_005"
	CodeUsernameExists   = "UVAL_006"
	CodeEmailExists      = "UVAL_007"
	CodePasswordEmpty    = "UVAL_008"
)
